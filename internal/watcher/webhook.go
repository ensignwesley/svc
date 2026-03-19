package watcher

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type webhookPayload struct {
	Event       string    `json:"event"`
	Service     string    `json:"service"`
	Description string    `json:"description,omitempty"`
	HealthURL   string    `json:"health_url"`
	Error       string    `json:"error,omitempty"`
	Failures    int       `json:"consecutive_failures,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
	PrevStatus  string    `json:"previous_status"`
	PrevChange  time.Time `json:"previous_change,omitempty"`
}

// deliverWebhook POSTs an event to the configured webhook URL with retry/backoff.
// Failures are logged to the delivery failure log; the watch loop is never blocked.
func deliverWebhook(cfg Config, ev Event, out io.Writer) {
	payload := webhookPayload{
		Event:       ev.Kind,
		Service:     ev.ServiceID,
		Description: ev.Description,
		HealthURL:   ev.HealthURL,
		Error:       ev.Error,
		Failures:    ev.Failures,
		Timestamp:   ev.Timestamp,
		PrevStatus:  ev.PrevStatus,
		PrevChange:  ev.PrevChange,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		logDeliveryFailure(cfg.DeliveryLogPath, ev.ServiceID, "marshal error: "+err.Error())
		return
	}

	backoffs := []time.Duration{5 * time.Second, 30 * time.Second, 2 * time.Minute}
	client := &http.Client{Timeout: 10 * time.Second}

	for attempt := 0; attempt <= len(backoffs); attempt++ {
		if attempt > 0 {
			fmt.Fprintf(out, "%s  → webhook retry %d for %s\n",
				timestamp(), attempt, ev.ServiceID)
			time.Sleep(backoffs[attempt-1])
		}

		resp, err := client.Post(cfg.WebhookURL, "application/json", bytes.NewReader(body))
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				fmt.Fprintf(out, "%s  → webhook OK (%d) for %s\n",
					timestamp(), resp.StatusCode, ev.ServiceID)
				return
			}
			err = fmt.Errorf("HTTP %d", resp.StatusCode)
		}

		if attempt == len(backoffs) {
			// Final attempt failed.
			msg := fmt.Sprintf("all retries failed for %s event on %s: %v",
				ev.Kind, ev.ServiceID, err)
			logDeliveryFailure(cfg.DeliveryLogPath, ev.ServiceID, msg)
			fmt.Fprintf(out, "%s  → webhook FAILED for %s (logged to %s)\n",
				timestamp(), ev.ServiceID, cfg.DeliveryLogPath)
		}
	}
}

// logDeliveryFailure appends a timestamped failure entry to the delivery log.
func logDeliveryFailure(path, service, msg string) {
	if path == "" {
		path = DefaultDeliveryFailurePath()
	}
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s  [%s] %s\n", time.Now().UTC().Format(time.RFC3339), service, msg)
}
