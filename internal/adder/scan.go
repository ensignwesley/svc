// Package adder — scan.go
// ScanFleet discovers all operator-installed systemd units, probes each one,
// and returns a list of ProbeResults. It skips units already present in the
// manifest to avoid duplicates.
package adder

import (
	"strings"

	"github.com/ensignwesley/svc/internal/checker"
)

// ScanResult wraps a ProbeResult with skip metadata.
type ScanResult struct {
	ProbeResult
	AlreadyInManifest bool
}

// ScanFleet discovers operator-installed systemd units and probes each one.
// knownIDs is the set of service IDs already in the manifest (may be nil).
// timeoutSec is the HTTP probe timeout per service.
func ScanFleet(knownIDs map[string]bool, timeoutSec int) ([]ScanResult, error) {
	units, err := checker.ListOperatorUnits()
	if err != nil {
		return nil, err
	}

	var results []ScanResult
	for _, unit := range units {
		id := unitToID(unit)

		// Check if already in manifest.
		alreadyKnown := knownIDs != nil && knownIDs[id]

		result := Probe(id, 0, timeoutSec)
		results = append(results, ScanResult{
			ProbeResult:       result,
			AlreadyInManifest: alreadyKnown,
		})
	}
	return results, nil
}

// unitToID strips the .service suffix and returns a clean ID.
func unitToID(unit string) string {
	return strings.TrimSuffix(unit, ".service")
}
