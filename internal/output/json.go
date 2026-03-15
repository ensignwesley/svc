package output

import (
	"encoding/json"
	"io"
	"time"
)

// StatusJSON is the machine-readable output for svc status.
type StatusJSON struct {
	CheckedAt string           `json:"checked_at"`
	Services  []ServiceStatus  `json:"services"`
}

// ServiceStatus is the status of a single service in JSON output.
type ServiceStatus struct {
	ID              string `json:"id"`
	Status          string `json:"status"` // "up" or "down"
	LatencyMs       int64  `json:"latency_ms"`
	VersionCurrent  string `json:"version_current,omitempty"`
	VersionLatest   string `json:"version_latest,omitempty"`
	VersionStatus   string `json:"version_status,omitempty"` // "current","behind","ahead","unknown",""
	SystemdActive   *bool  `json:"systemd_active,omitempty"`
}

// CheckJSON is the machine-readable output for svc check.
type CheckJSON struct {
	CheckedAt    string           `json:"checked_at"`
	Services     []ServiceStatus  `json:"services"`
	Undocumented []string         `json:"undocumented"`
	DriftCount   int              `json:"drift_count"`
	ExitCode     int              `json:"exit_code"`
}

// WriteJSON encodes v as pretty JSON to w.
func WriteJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// Now returns the current UTC time in RFC3339 format.
func Now() string {
	return time.Now().UTC().Format(time.RFC3339)
}
