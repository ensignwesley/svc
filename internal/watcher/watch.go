package watcher

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"sort"
	"syscall"
	"time"

	"github.com/ensignwesley/svc/internal/checker"
	"github.com/ensignwesley/svc/internal/manifest"
)

// Event represents a state change for a service.
type Event struct {
	Kind        string    // "down" or "up"
	ServiceID   string
	Description string
	HealthURL   string
	Error       string
	Failures    int
	Timestamp   time.Time
	PrevStatus  string
	PrevChange  time.Time
}

// Config holds svc watch runtime configuration.
type Config struct {
	ManifestPath    string
	StatePath       string
	WebhookURL      string
	Interval        int // seconds
	FailThreshold   int
	TimeoutSec      int
	NoSystemd       bool
	Stdout          bool
	DeliveryLogPath string
}

// Watch runs the main polling loop until SIGTERM/SIGINT.
func Watch(cfg Config, out io.Writer) error {
	m, err := manifest.Load(cfg.ManifestPath)
	if err != nil {
		return err
	}

	state, err := Load(cfg.StatePath)
	if err != nil {
		return fmt.Errorf("loading state: %w", err)
	}

	fmt.Fprintf(out, "%s  svc watch starting — %d services, %ds interval\n",
		timestamp(), len(m.Services), cfg.Interval)

	// Handle graceful shutdown.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	ticker := time.NewTicker(time.Duration(cfg.Interval) * time.Second)
	defer ticker.Stop()

	// Run first check immediately.
	runCheck(cfg, m, state, out)
	if err := Save(cfg.StatePath, state); err != nil {
		fmt.Fprintf(out, "%s  ⚠️  state save failed: %v\n", timestamp(), err)
	}

	for {
		select {
		case <-stop:
			fmt.Fprintf(out, "%s  svc watch stopping\n", timestamp())
			return nil
		case <-ticker.C:
			runCheck(cfg, m, state, out)
			if err := Save(cfg.StatePath, state); err != nil {
				fmt.Fprintf(out, "%s  ⚠️  state save failed: %v\n", timestamp(), err)
			}
		}
	}
}

// runCheck polls all services and updates state, firing events on transitions.
func runCheck(cfg Config, m *manifest.Manifest, state *WatchState, out io.Writer) {
	// Build health check targets.
	targets := make(map[string]string)
	for id, svc := range m.Services {
		targets[id] = manifest.ResolveHealthURL(m, svc)
	}

	results := checker.CheckAllHealth(targets, cfg.TimeoutSec)

	// Sort for stable output.
	sort.Slice(results, func(i, j int) bool {
		return results[i].ServiceID < results[j].ServiceID
	})

	for _, r := range results {
		svc := m.Services[r.ServiceID]
		ss := state.GetOrInit(r.ServiceID)
		now := time.Now().UTC()

		prevStatus := ss.Status

		if r.Up {
			wasDown := ss.Status == StatusDown || ss.Status == StatusDegraded

			// Reset failure count, mark up.
			ss.ConsecutiveFailures = 0
			ss.LastCheck = now
			ss.LastError = ""

			if ss.Status != StatusUp {
				ss.LastChange = now
			}
			ss.Status = StatusUp

			if wasDown && ss.Alerted {
				// Recovery event.
				ss.Alerted = false
				ev := Event{
					Kind:        "up",
					ServiceID:   r.ServiceID,
					Description: svc.Description,
					HealthURL:   manifest.ResolveHealthURL(m, svc),
					Timestamp:   now,
					PrevStatus:  prevStatus.String(),
					PrevChange:  ss.LastChange,
				}
				fmt.Fprintf(out, "%s  %-16s ✅ up (%dms) → RECOVERY\n",
					timestamp(), r.ServiceID, r.LatencyMS)
				fireEvent(cfg, ev, out)
			} else {
				fmt.Fprintf(out, "%s  %-16s ✅ up (%dms)\n",
					timestamp(), r.ServiceID, r.LatencyMS)
			}
		} else {
			// Failure.
			ss.ConsecutiveFailures++
			ss.LastCheck = now
			ss.LastError = r.Err

			if ss.ConsecutiveFailures >= cfg.FailThreshold && !ss.Alerted {
				// Threshold crossed — alert.
				if ss.Status != StatusDown {
					ss.LastChange = now
				}
				ss.Status = StatusDown
				ss.Alerted = true

				ev := Event{
					Kind:        "down",
					ServiceID:   r.ServiceID,
					Description: svc.Description,
					HealthURL:   manifest.ResolveHealthURL(m, svc),
					Error:       r.Err,
					Failures:    ss.ConsecutiveFailures,
					Timestamp:   now,
					PrevStatus:  prevStatus.String(),
					PrevChange:  ss.LastChange,
				}
				fmt.Fprintf(out, "%s  %-16s ❌ down — %s (failure %d/%d) → ALERT\n",
					timestamp(), r.ServiceID, r.Err,
					ss.ConsecutiveFailures, cfg.FailThreshold)
				fireEvent(cfg, ev, out)
			} else if ss.ConsecutiveFailures < cfg.FailThreshold {
				// Degraded — not yet at threshold.
				ss.Status = StatusDegraded
				fmt.Fprintf(out, "%s  %-16s ❌ down — %s (failure %d/%d)\n",
					timestamp(), r.ServiceID, r.Err,
					ss.ConsecutiveFailures, cfg.FailThreshold)
			} else {
				// Already alerted, still down.
				fmt.Fprintf(out, "%s  %-16s ❌ down — %s (still down, failure %d)\n",
					timestamp(), r.ServiceID, r.Err, ss.ConsecutiveFailures)
			}
		}
	}
}

// fireEvent dispatches an event to stdout and/or webhook.
func fireEvent(cfg Config, ev Event, out io.Writer) {
	if cfg.WebhookURL != "" {
		go deliverWebhook(cfg, ev, out)
	}
}

func timestamp() string {
	return time.Now().UTC().Format("2006-01-02 15:04:05")
}
