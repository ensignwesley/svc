package checker

import (
	"os/exec"
	"strings"
)

// SystemdResult holds the result of a systemd unit check.
type SystemdResult struct {
	Unit      string
	Active    bool
	LoadState string // "loaded", "not-found", "masked", etc.
	Err       string
}

// CheckUnit verifies a systemd unit is active and loaded.
// It tries the user session first, then falls back to the system session.
func CheckUnit(unit string) SystemdResult {
	// Try user session first (most self-hosted services run as user units).
	result := checkUnitScope(unit, true)
	if result.LoadState != "not-found" && result.Err != "systemctl unavailable" {
		return result
	}
	// Fall back to system session.
	return checkUnitScope(unit, false)
}

func checkUnitScope(unit string, userSession bool) SystemdResult {
	result := SystemdResult{Unit: unit}

	args := []string{"show", "--property=LoadState"}
	if userSession {
		args = append([]string{"--user"}, args...)
	}
	args = append(args, unit)

	loadOut, err := exec.Command("systemctl", args...).Output()
	if err != nil {
		result.Err = "systemctl unavailable"
		return result
	}
	loadState := parseProperty(string(loadOut), "LoadState")
	result.LoadState = loadState

	if loadState == "not-found" {
		result.Active = false
		result.Err = "unit not found: " + unit
		return result
	}

	// Check active state.
	activeArgs := []string{"show", "--property=ActiveState"}
	if userSession {
		activeArgs = append([]string{"--user"}, activeArgs...)
	}
	activeArgs = append(activeArgs, unit)

	activeOut, err := exec.Command("systemctl", activeArgs...).Output()
	if err != nil {
		result.Err = "systemctl error"
		return result
	}
	activeState := parseProperty(string(activeOut), "ActiveState")
	result.Active = activeState == "active"
	if !result.Active {
		result.Err = "unit " + activeState
	}

	return result
}

// SystemdAvailable returns true if systemctl is available on this system.
func SystemdAvailable() bool {
	_, err := exec.LookPath("systemctl")
	return err == nil
}

// ListOperatorUnits returns units from /etc/systemd/system/ and
// ~/.config/systemd/user/ — units the operator installed, not the OS.
func ListOperatorUnits() ([]string, error) {
	seen := make(map[string]bool)
	var operatorUnits []string

	for _, scope := range []struct {
		userFlag bool
		label    string
	}{
		{false, "system"},
		{true, "user"},
	} {
		args := []string{"list-unit-files", "--type=service", "--no-legend", "--no-pager"}
		if scope.userFlag {
			args = append([]string{"--user"}, args...)
		}
		out, err := exec.Command("systemctl", args...).Output()
		if err != nil {
			continue
		}

		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) == 0 {
				continue
			}
			unitName := fields[0]
			if !strings.HasSuffix(unitName, ".service") || seen[unitName] {
				continue
			}

			// Get fragment path to determine provenance.
			fragArgs := []string{"show", "--property=FragmentPath", unitName}
			if scope.userFlag {
				fragArgs = append([]string{"--user"}, fragArgs...)
			}
			fragOut, err := exec.Command("systemctl", fragArgs...).Output()
			if err != nil {
				continue
			}
			fragPath := parseProperty(string(fragOut), "FragmentPath")
			if isOperatorPath(fragPath) {
				operatorUnits = append(operatorUnits, unitName)
				seen[unitName] = true
			}
		}
	}

	return operatorUnits, nil
}

// isOperatorPath returns true for unit files installed by the operator
// (not by the package manager).
func isOperatorPath(path string) bool {
	if path == "" {
		return false
	}
	// /etc/systemd/system/ — system-level operator units
	if strings.HasPrefix(path, "/etc/systemd/system/") {
		return true
	}
	// ~/.config/systemd/user/ — user-level operator units
	if strings.Contains(path, "/.config/systemd/user/") {
		return true
	}
	return false
}

// parseProperty extracts the value from "Key=Value\n" output.
func parseProperty(output, key string) string {
	prefix := key + "="
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimPrefix(line, prefix)
		}
	}
	return ""
}
