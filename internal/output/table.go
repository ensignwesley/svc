package output

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/ensignwesley/svc/internal/checker"
)

// isTTY returns true if stdout is a terminal.
func isTTY() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
)

func colorize(color, s string) string {
	if !isTTY() {
		return s
	}
	return color + s + colorReset
}

// PrintStatusTable writes a formatted health status table to w.
func PrintStatusTable(w io.Writer, results []checker.HealthResult) {
	// Sort by service ID for stable output.
	sorted := make([]checker.HealthResult, len(results))
	copy(sorted, results)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].ServiceID < sorted[j].ServiceID
	})

	// Calculate column widths.
	nameWidth := len("Service")
	for _, r := range sorted {
		if len(r.ServiceID) > nameWidth {
			nameWidth = len(r.ServiceID)
		}
	}

	// Header.
	header := fmt.Sprintf("  %-*s  %-10s  %-8s  %s",
		nameWidth, "Service", "Status", "Latency", "Note")
	sep := "  " + strings.Repeat("─", len(header)-2)
	fmt.Fprintln(w, header)
	fmt.Fprintln(w, sep)

	// Rows.
	for _, r := range sorted {
		var status, latency, note string

		if r.Up {
			status = colorize(colorGreen, "✅ up")
			latency = fmt.Sprintf("%dms", r.LatencyMS)
		} else {
			status = colorize(colorRed, "❌ down")
			latency = "—"
			note = r.Err
		}

		fmt.Fprintf(w, "  %-*s  %-10s  %-8s  %s\n",
			nameWidth, r.ServiceID, status, latency, note)
	}
}
