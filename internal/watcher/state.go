package watcher

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// ServiceStatus represents the current known state of a service.
type ServiceStatus int

const (
	StatusUnknown ServiceStatus = iota
	StatusUp
	StatusDegraded // failing but below alert threshold
	StatusDown     // at or above threshold, alert fired
)

func (s ServiceStatus) String() string {
	switch s {
	case StatusUp:
		return "up"
	case StatusDegraded:
		return "degraded"
	case StatusDown:
		return "down"
	default:
		return "unknown"
	}
}

// ServiceState tracks the watch state for a single service.
type ServiceState struct {
	Status              ServiceStatus `json:"status"`
	ConsecutiveFailures int           `json:"consecutive_failures"`
	Alerted             bool          `json:"alerted"`
	LastCheck           time.Time     `json:"last_check"`
	LastChange          time.Time     `json:"last_change"`
	LastError           string        `json:"last_error,omitempty"`
}

// WatchState is the full persisted state for svc watch.
type WatchState struct {
	Services map[string]*ServiceState `json:"services"`
}

// NewWatchState returns an empty watch state.
func NewWatchState() *WatchState {
	return &WatchState{
		Services: make(map[string]*ServiceState),
	}
}

// GetOrInit returns the state for a service, initialising if not present.
func (ws *WatchState) GetOrInit(id string) *ServiceState {
	if _, ok := ws.Services[id]; !ok {
		ws.Services[id] = &ServiceState{Status: StatusUnknown}
	}
	return ws.Services[id]
}

// Load reads watch state from disk. Returns empty state if file doesn't exist.
func Load(path string) (*WatchState, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return NewWatchState(), nil
	}
	if err != nil {
		return nil, err
	}
	var ws WatchState
	if err := json.Unmarshal(data, &ws); err != nil {
		return NewWatchState(), nil // corrupt state: start fresh
	}
	if ws.Services == nil {
		ws.Services = make(map[string]*ServiceState)
	}
	return &ws, nil
}

// Save writes watch state to disk atomically.
func Save(path string, ws *WatchState) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(ws, "", "  ")
	if err != nil {
		return err
	}
	// Write to temp file then rename for atomicity.
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// DefaultStatePath returns the default state file location.
func DefaultStatePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".svc-watch-state.json"
	}
	return filepath.Join(home, ".local", "share", "svc", "watch-state.json")
}

// DefaultDeliveryFailurePath returns the delivery failure log path.
func DefaultDeliveryFailurePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".svc-delivery-failures.log"
	}
	return filepath.Join(home, ".local", "share", "svc", "delivery-failures.log")
}
