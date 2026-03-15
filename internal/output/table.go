package output

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorDim    = "\033[2m"
)

// IsTTY returns true if stdout is a terminal.
func IsTTY() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// Row is one line of the status table.
type Row struct {
	ID         string
	Status     string // "up", "down"
	LatencyMs  int64
	Version    string // "current", "behind", "ahead", "unknown", ""
	VersionStr string // e.g. "⚠️ behind (latest: 1.3.0)"
	Note       string
}

// PrintTable writes a status table to w.
func PrintTable(w io.Writer, rows []Row, color bool) {
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].ID < rows[j].ID
	})

	// Column widths
	colID := 14
	for _, r := range rows {
		if len(r.ID) > colID {
			colID = len(r.ID)
		}
	}

	header := fmt.Sprintf("%-*s  %-8s  %-10s  %s", colID, "Service", "Status", "Latency", "Note")
	sep := strings.Repeat("─", len(header)+4)

	fmt.Fprintln(w, header)
	fmt.Fprintln(w, sep)

	for _, r := range rows {
		statusStr := r.Status
		latencyStr := "—"
		if r.LatencyMs > 0 || r.Status == "up" {
			latencyStr = fmt.Sprintf("%dms", r.LatencyMs)
		}

		if color {
			switch r.Status {
			case "up":
				statusStr = colorGreen + "✓ up" + colorReset
			case "down":
				statusStr = colorRed + "✗ down" + colorReset
			}
		} else {
			switch r.Status {
			case "up":
				statusStr = "✓ up"
			case "down":
				statusStr = "✗ down"
			}
		}

		note := r.Note
		if r.VersionStr != "" {
			if note != "" {
				note += "  " + r.VersionStr
			} else {
				note = r.VersionStr
			}
		}

		if color && strings.Contains(note, "behind") {
			note = colorYellow + note + colorReset
		}

		fmt.Fprintf(w, "%-*s  %-8s  %-10s  %s\n", colID, r.ID, statusStr, latencyStr, note)
	}
}

// PrintDrift writes a drift report to w.
type DriftItem struct {
	ID      string
	Kind    string // "down", "unit-inactive", "version-behind", "undocumented"
	Detail  string
}

func PrintDrift(w io.Writer, items []DriftItem, undocumented []string, color bool) {
	total := len(items) + len(undocumented)
	if total == 0 {
		msg := "No drift detected. All services match the manifest."
		if color {
			msg = colorGreen + "✓ " + colorReset + msg
		}
		fmt.Fprintln(w, msg)
		return
	}

	for _, item := range items {
		switch item.Kind {
		case "down":
			prefix := "❌"
			if color {
				prefix = colorRed + "✗" + colorReset
			}
			fmt.Fprintf(w, "  %-20s %s  health endpoint unreachable%s\n",
				item.ID, prefix, maybeDetail(item.Detail))
		case "unit-inactive":
			prefix := "❌"
			if color {
				prefix = colorRed + "✗" + colorReset
			}
			fmt.Fprintf(w, "  %-20s %s  systemd unit not active (%s)\n",
				item.ID, prefix, item.Detail)
		case "version-behind":
			prefix := "⚠️"
			if color {
				prefix = colorYellow + "⚠" + colorReset
			}
			fmt.Fprintf(w, "  %-20s %s  version behind: %s\n",
				item.ID, prefix, item.Detail)
		}
	}

	if len(undocumented) > 0 {
		fmt.Fprintln(w, "\nUndocumented units:")
		for _, u := range undocumented {
			prefix := "⚠️"
			if color {
				prefix = colorYellow + "⚠" + colorReset
			}
			fmt.Fprintf(w, "  %s  %s — active, no manifest entry\n", prefix, u)
		}
	}

	// Summary
	down := 0
	behind := 0
	for _, item := range items {
		switch item.Kind {
		case "down", "unit-inactive":
			down++
		case "version-behind":
			behind++
		}
	}

	parts := []string{}
	if down > 0 {
		s := fmt.Sprintf("%d down", down)
		if color {
			s = colorRed + s + colorReset
		}
		parts = append(parts, s)
	}
	if behind > 0 {
		s := fmt.Sprintf("%d behind", behind)
		if color {
			s = colorYellow + s + colorReset
		}
		parts = append(parts, s)
	}
	if len(undocumented) > 0 {
		s := fmt.Sprintf("%d undocumented", len(undocumented))
		if color {
			s = colorYellow + s + colorReset
		}
		parts = append(parts, s)
	}

	fmt.Fprintf(w, "\nSummary: %s\n", strings.Join(parts, ", "))
}

func maybeDetail(d string) string {
	if d == "" {
		return ""
	}
	return " (" + d + ")"
}
