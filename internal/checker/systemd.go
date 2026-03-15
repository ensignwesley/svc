package checker

import (
	"os/exec"
	"strings"
)

// SystemdResult holds the result of a systemctl is-active check.
type SystemdResult struct {
	Unit   string
	Active bool
	State  string // "active", "inactive", "failed", etc.
}

// CheckSystemd checks if a systemd unit is active.
// Returns ok=false (with no error) on non-systemd systems.
func CheckSystemd(unit string) SystemdResult {
	out, err := exec.Command("systemctl", "--user", "is-active", unit).Output()
	if err != nil {
		// Also try system-level
		out2, err2 := exec.Command("systemctl", "is-active", unit).Output()
		if err2 != nil {
			return SystemdResult{Unit: unit, Active: false, State: "unknown"}
		}
		out = out2
	}
	state := strings.TrimSpace(string(out))
	return SystemdResult{Unit: unit, Active: state == "active", State: state}
}

// ListUserUnits returns all active user-level systemd service units.
func ListUserUnits() ([]string, error) {
	out, err := exec.Command("systemctl", "--user", "--no-legend", "--state=active",
		"list-units", "--type=service").Output()
	if err != nil {
		return nil, err
	}

	var units []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 1 && strings.HasSuffix(fields[0], ".service") {
			units = append(units, fields[0])
		}
	}
	return units, nil
}
