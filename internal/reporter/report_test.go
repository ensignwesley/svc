package reporter

import (
	"strings"
	"testing"
	"time"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{7 * 24 * time.Hour, "7d"},
		{30 * 24 * time.Hour, "30d"},
		{24 * time.Hour, "1d"},
		{12 * time.Hour, "12h"},
		{1 * time.Hour, "1h"},
		{90 * time.Minute, "1h30m0s"}, // not cleanly expressible, falls through
	}
	for _, tc := range tests {
		got := FormatDuration(tc.d)
		if got != tc.want {
			t.Errorf("FormatDuration(%v) = %q, want %q", tc.d, got, tc.want)
		}
	}
}

func TestAvgUptimePct_Empty(t *testing.T) {
	r := Report{}
	if r.AvgUptimePct() != 100.0 {
		t.Errorf("empty report: AvgUptimePct() = %f, want 100.0", r.AvgUptimePct())
	}
}

func TestAvgUptimePct(t *testing.T) {
	r := Report{
		Services: []ServiceReport{
			{ID: "a", UptimePct: 100.0},
			{ID: "b", UptimePct: 90.0},
		},
	}
	got := r.AvgUptimePct()
	want := 95.0
	if got != want {
		t.Errorf("AvgUptimePct() = %f, want %f", got, want)
	}
}

func TestPrintTable_NoData(t *testing.T) {
	var buf strings.Builder
	r := Report{
		GeneratedAt: time.Now(),
		Window:      7 * 24 * time.Hour,
	}
	PrintTable(&buf, r)
	if !strings.Contains(buf.String(), "No data") {
		t.Errorf("expected 'No data' message, got:\n%s", buf.String())
	}
}

func TestPrintTable_WithData(t *testing.T) {
	var buf strings.Builder
	dur := int64(480)
	r := Report{
		GeneratedAt: time.Now(),
		Window:      7 * 24 * time.Hour,
		Services: []ServiceReport{
			{ID: "blog", UptimePct: 100.0, Incidents: 0},
			{ID: "dead-drop", UptimePct: 99.2, Incidents: 1, LastIncident: &IncidentSummary{
				StartedAt:   time.Date(2026, 3, 21, 2, 14, 0, 0, time.UTC),
				DurationSec: &dur,
			}},
		},
	}
	PrintTable(&buf, r)
	out := buf.String()

	if !strings.Contains(out, "blog") {
		t.Error("expected 'blog' in output")
	}
	if !strings.Contains(out, "99.2%") {
		t.Error("expected '99.2%' in output")
	}
	if !strings.Contains(out, "Mar 21 02:14") {
		t.Error("expected incident timestamp in output")
	}
	if !strings.Contains(out, "Summary:") {
		t.Error("expected summary line in output")
	}
}

func TestPrintMarkdown_NoData(t *testing.T) {
	var buf strings.Builder
	r := Report{
		GeneratedAt: time.Now(),
		Window:      7 * 24 * time.Hour,
	}
	PrintMarkdown(&buf, r)
	if !strings.Contains(buf.String(), "No data") {
		t.Errorf("expected 'No data' message, got:\n%s", buf.String())
	}
}

func TestPrintMarkdown_WithData(t *testing.T) {
	var buf strings.Builder
	r := Report{
		GeneratedAt: time.Now(),
		Window:      7 * 24 * time.Hour,
		Services: []ServiceReport{
			{ID: "blog", UptimePct: 100.0, Incidents: 0},
		},
	}
	PrintMarkdown(&buf, r)
	out := buf.String()

	if !strings.Contains(out, "## Fleet Report") {
		t.Error("expected markdown header")
	}
	if !strings.Contains(out, "| blog |") {
		t.Error("expected table row for blog")
	}
	if !strings.Contains(out, "100.0%") {
		t.Error("expected uptime percentage")
	}
}
