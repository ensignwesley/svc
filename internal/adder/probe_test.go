package adder_test

import (
	"strings"
	"testing"

	"github.com/ensignwesley/svc/internal/adder"
)

func TestScaffoldMinimal(t *testing.T) {
	r := adder.ProbeResult{
		ID:   "my-service",
		Port: 8080,
	}
	out := adder.Scaffold(r, "2026-03-20")

	if !strings.Contains(out, "my-service:") {
		t.Error("expected service id in output")
	}
	if !strings.Contains(out, "port: 8080") {
		t.Error("expected port in output")
	}
	if !strings.Contains(out, "2026-03-20") {
		t.Error("expected added date in output")
	}
}

func TestScaffoldWithUnit(t *testing.T) {
	r := adder.ProbeResult{
		ID:          "dead-drop",
		Port:        3001,
		SystemdUnit: "dead-drop.service",
		UnitActive:  true,
	}
	out := adder.Scaffold(r, "2026-03-20")

	if !strings.Contains(out, `systemd_unit: "dead-drop.service"`) {
		t.Errorf("expected systemd_unit in output, got:\n%s", out)
	}
}

func TestScaffoldNonStandardHealthURL(t *testing.T) {
	r := adder.ProbeResult{
		ID:        "forth",
		Port:      3005,
		HealthURL: "http://localhost:3005/",
		HealthOK:  true,
	}
	out := adder.Scaffold(r, "2026-03-20")

	// Non-standard path should emit explicit health_url.
	if !strings.Contains(out, "health_url:") {
		t.Errorf("expected health_url for non-standard path, got:\n%s", out)
	}
}

func TestScaffoldStandardHealthURL(t *testing.T) {
	r := adder.ProbeResult{
		ID:        "my-api",
		Port:      8080,
		HealthURL: "http://localhost:8080/health",
		HealthOK:  true,
	}
	out := adder.Scaffold(r, "2026-03-20")

	// Standard /health path should NOT emit redundant health_url.
	if strings.Contains(out, "health_url:") {
		t.Errorf("unexpected health_url for standard /health path, got:\n%s", out)
	}
}

func TestScaffoldWithNotes(t *testing.T) {
	r := adder.ProbeResult{
		ID:    "mystery-service",
		Port:  9999,
		Notes: []string{"could not detect port — set port or health_url manually"},
	}
	out := adder.Scaffold(r, "2026-03-20")

	if !strings.Contains(out, "# Notes from probe:") {
		t.Errorf("expected notes section, got:\n%s", out)
	}
	if !strings.Contains(out, "could not detect port") {
		t.Errorf("expected note text, got:\n%s", out)
	}
}
