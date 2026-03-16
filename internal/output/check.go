package output

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/ensignwesley/svc/internal/checker"
	"github.com/ensignwesley/svc/internal/manifest"
)

// PrintCheckTable prints the drift check results and returns the drift count.
func PrintCheckTable(w io.Writer, healthResults []checker.HealthResult, systemdResults map[string]checker.SystemdResult, services map[string]manifest.Service) int {
	// Sort by service ID.
	sorted := make([]checker.HealthResult, len(healthResults))
	copy(sorted, healthResults)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].ServiceID < sorted[j].ServiceID
	})

	// Column widths.
	nameWidth := len("Service")
	for _, r := range sorted {
		if len(r.ServiceID) > nameWidth {
			nameWidth = len(r.ServiceID)
		}
	}

	header := fmt.Sprintf("  %-*s  %-10s  %-8s  %s",
		nameWidth, "Service", "Health", "Latency", "Notes")
	sep := "  " + strings.Repeat("─", len(header)-2)
	fmt.Fprintln(w, header)
	fmt.Fprintln(w, sep)

	driftCount := 0

	for _, r := range sorted {
		var healthStr, latency, notes string

		if r.Up {
			healthStr = colorize(colorGreen, "✅ up")
			latency = fmt.Sprintf("%dms", r.LatencyMS)
		} else {
			healthStr = colorize(colorRed, "❌ down")
			latency = "—"
			notes = r.Err
			driftCount++
		}

		// Append systemd note if available.
		if sd, ok := systemdResults[r.ServiceID]; ok {
			if !sd.Active {
				sdNote := colorize(colorYellow, "systemd: "+sd.Err)
				if notes != "" {
					notes += "; " + sdNote
				} else {
					notes = sdNote
				}
				if r.Up {
					// Service responds but unit is not active — still drift.
					driftCount++
				}
			}
		}

		fmt.Fprintf(w, "  %-*s  %-10s  %-8s  %s\n",
			nameWidth, r.ServiceID, healthStr, latency, notes)
	}

	return driftCount
}
