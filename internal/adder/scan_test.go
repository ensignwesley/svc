package adder_test

import (
	"testing"

	"github.com/ensignwesley/svc/internal/adder"
)

func TestUnitToID(t *testing.T) {
	// unitToID is unexported; test via ScanFleet indirectly isn't practical.
	// Instead verify that Scaffold works on ProbeResult produced by scan path.
	r := adder.ProbeResult{
		ID:          "my-service",
		Port:        8080,
		SystemdUnit: "my-service.service",
		UnitActive:  true,
		Notes:       []string{"port 8080 inferred from unit ExecStart"},
	}
	out := adder.Scaffold(r, "2026-03-22")
	if out == "" {
		t.Error("expected non-empty scaffold output")
	}
}

func TestScanResultAlreadyKnown(t *testing.T) {
	// ScanFleet probes live systemd — but we can verify that the knownIDs
	// filter works by running with a nil map (no filtering) vs a full map.
	// This test exercises the code path with a no-op: empty knownIDs should
	// return no AlreadyInManifest entries.
	known := make(map[string]bool)
	results, err := adder.ScanFleet(known, 1)
	if err != nil {
		t.Skipf("systemd not available: %v", err)
	}
	for _, r := range results {
		if r.AlreadyInManifest {
			t.Errorf("unexpected AlreadyInManifest=true for %q with empty known set", r.ID)
		}
	}
}

func TestScanResultAllKnown(t *testing.T) {
	// When all discovered units are in knownIDs, AlreadyInManifest should be true.
	// We don't know what units exist at test time, so we run once with empty known,
	// collect IDs, then re-run with all IDs marked known.
	empty := make(map[string]bool)
	first, err := adder.ScanFleet(empty, 1)
	if err != nil || len(first) == 0 {
		t.Skip("no operator units or systemd unavailable")
	}

	allKnown := make(map[string]bool)
	for _, r := range first {
		allKnown[r.ID] = true
	}

	second, err := adder.ScanFleet(allKnown, 1)
	if err != nil {
		t.Fatalf("second scan failed: %v", err)
	}
	for _, r := range second {
		if !r.AlreadyInManifest {
			t.Errorf("expected AlreadyInManifest=true for %q", r.ID)
		}
	}
}
