package checker

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// RemoteSystemdResult holds the result of a remote systemd unit check via SSH.
type RemoteSystemdResult struct {
	Host        string
	Unit        string
	Active      bool
	LoadState   string
	Err         string
	LatencyMS   int64
}

// CheckRemoteUnit verifies a systemd unit on a remote host via SSH.
// Uses ~/.ssh/config for auth — no credentials in code.
// SSH failures return a result with Err set, not a Go error.
func CheckRemoteUnit(host, unit string, timeoutSec int) RemoteSystemdResult {
	result := RemoteSystemdResult{Host: host, Unit: unit}

	// Run systemctl show over SSH. BatchMode=yes prevents interactive prompts.
	cmd := exec.Command(
		"ssh",
		"-o", "BatchMode=yes",
		"-o", fmt.Sprintf("ConnectTimeout=%d", timeoutSec),
		"-o", "StrictHostKeyChecking=accept-new",
		host,
		"systemctl", "--user", "show", "--property=LoadState,ActiveState", unit,
	)

	start := time.Now()
	out, err := cmd.Output()
	result.LatencyMS = time.Since(start).Milliseconds()

	if err != nil {
		// Try system session if user session fails.
		cmd2 := exec.Command(
			"ssh",
			"-o", "BatchMode=yes",
			"-o", fmt.Sprintf("ConnectTimeout=%d", timeoutSec),
			"-o", "StrictHostKeyChecking=accept-new",
			host,
			"systemctl", "show", "--property=LoadState,ActiveState", unit,
		)
		out2, err2 := cmd2.Output()
		if err2 != nil {
			result.Err = summariseSSHError(err.Error())
			return result
		}
		out = out2
	}

	loadState := parseProperty(string(out), "LoadState")
	result.LoadState = loadState

	if loadState == "not-found" {
		result.Active = false
		result.Err = "unit not found on " + host + ": " + unit
		return result
	}

	activeState := parseProperty(string(out), "ActiveState")
	result.Active = activeState == "active"
	if !result.Active {
		result.Err = "unit " + activeState + " on " + host
	}

	return result
}

// IsRemoteHost returns true if the host string refers to a remote machine.
func IsRemoteHost(host string) bool {
	if host == "" || host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return false
	}
	return true
}

// summariseSSHError converts SSH error messages into operator-readable form.
func summariseSSHError(msg string) string {
	if strings.Contains(msg, "Connection refused") {
		return "SSH connection refused"
	}
	if strings.Contains(msg, "No route to host") || strings.Contains(msg, "Network is unreachable") {
		return "SSH host unreachable"
	}
	if strings.Contains(msg, "Connection timed out") || strings.Contains(msg, "timed out") {
		return "SSH timeout"
	}
	if strings.Contains(msg, "Permission denied") || strings.Contains(msg, "publickey") {
		return "SSH auth failed (check ~/.ssh/config)"
	}
	if strings.Contains(msg, "Host key verification failed") {
		return "SSH host key mismatch"
	}
	return "SSH error: " + msg
}
