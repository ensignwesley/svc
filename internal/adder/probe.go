// Package adder probes a running service and produces a manifest entry scaffold.
package adder

import (
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

// ProbeResult holds everything discovered about a service candidate.
type ProbeResult struct {
	ID          string
	Port        int
	HealthURL   string
	HealthOK    bool
	SystemdUnit string
	UnitActive  bool
	Notes       []string
}

// Probe discovers what it can about a service by ID and optional port hint.
func Probe(id string, portHint int, timeoutSec int) ProbeResult {
	r := ProbeResult{ID: id}

	// 1. Systemd unit detection — try id.service first, then bare id.
	candidates := []string{id + ".service", id}
	for _, unit := range candidates {
		if active, found := checkUnit(unit); found {
			r.SystemdUnit = unit
			r.UnitActive = active
			break
		}
	}

	// 2. Port detection.
	if portHint > 0 {
		r.Port = portHint
	} else if r.SystemdUnit != "" {
		// Try to read port from systemd unit's ExecStart line.
		if p := inferPortFromUnit(r.SystemdUnit); p > 0 {
			r.Port = p
			r.Notes = append(r.Notes, fmt.Sprintf("port %d inferred from unit ExecStart", p))
		}
	}

	// 3. Health URL probe.
	if r.Port > 0 {
		// Try /health first, fall back to / 
		for _, path := range []string{"/healthz", "/health", "/ping", "/"} {
			url := fmt.Sprintf("http://localhost:%d%s", r.Port, path)
			if ok, latency := probeHTTP(url, timeoutSec); ok {
				r.HealthURL = url
				r.HealthOK = true
				r.Notes = append(r.Notes, fmt.Sprintf("health endpoint responds at %s (%dms)", path, latency))
				break
			}
		}
		if !r.HealthOK {
			r.Notes = append(r.Notes, fmt.Sprintf("no health endpoint found on port %d — set health_url manually", r.Port))
		}
	} else {
		r.Notes = append(r.Notes, "could not detect port — set port or health_url manually")
	}

	return r
}

// Scaffold generates a YAML manifest entry from a ProbeResult.
// Output goes to stdout — the caller appends it to services.yaml.
func Scaffold(r ProbeResult, today string) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("  %s:\n", r.ID))
	b.WriteString(fmt.Sprintf("    description: \"%s — describe this service\"\n", r.ID))

	if r.Port > 0 {
		b.WriteString(fmt.Sprintf("    port: %d\n", r.Port))
	}

	// If health URL is non-standard path, emit it explicitly.
	if r.HealthURL != "" && !strings.HasSuffix(r.HealthURL, "/health") {
		b.WriteString(fmt.Sprintf("    health_url: %q\n", r.HealthURL))
	} else if r.Port == 0 && r.HealthURL != "" {
		b.WriteString(fmt.Sprintf("    health_url: %q\n", r.HealthURL))
	}

	if r.SystemdUnit != "" {
		b.WriteString(fmt.Sprintf("    systemd_unit: %q\n", r.SystemdUnit))
	}

	b.WriteString(fmt.Sprintf("    added: %q\n", today))

	if len(r.Notes) > 0 {
		b.WriteString("    # Notes from probe:\n")
		for _, n := range r.Notes {
			b.WriteString(fmt.Sprintf("    #   %s\n", n))
		}
	}

	return b.String()
}

// checkUnit returns (active, found) for a systemd unit in user or system session.
func checkUnit(unit string) (active bool, found bool) {
	for _, scope := range [][]string{{"--user"}, {}} {
		args := append(scope, "show", "--property=LoadState,ActiveState", unit)
		out, err := exec.Command("systemctl", args...).Output()
		if err != nil {
			continue
		}
		loadState := parseProperty(string(out), "LoadState")
		if loadState == "not-found" || loadState == "" {
			continue
		}
		activeState := parseProperty(string(out), "ActiveState")
		return activeState == "active", true
	}
	return false, false
}

// inferPortFromUnit attempts to read a port number from a systemd unit's ExecStart.
func inferPortFromUnit(unit string) int {
	for _, scope := range [][]string{{"--user"}, {}} {
		args := append(scope, "show", "--property=ExecStart", unit)
		out, err := exec.Command("systemctl", args...).Output()
		if err != nil {
			continue
		}
		// Look for --port N or -p N patterns in ExecStart value.
		line := string(out)
		for _, prefix := range []string{"--port=", "--port ", "-p ", "-p="} {
			if idx := strings.Index(line, prefix); idx >= 0 {
				rest := line[idx+len(prefix):]
				var port int
				if n, _ := fmt.Sscanf(rest, "%d", &port); n == 1 && port > 0 {
					return port
				}
			}
		}
	}
	return 0
}

// probeHTTP returns (ok, latencyMs) for a GET request.
func probeHTTP(url string, timeoutSec int) (bool, int64) {
	client := &http.Client{Timeout: time.Duration(timeoutSec) * time.Second}
	start := time.Now()
	resp, err := client.Get(url)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return false, 0
	}
	resp.Body.Close()
	return resp.StatusCode == 200, latency
}

// parseProperty extracts Key=Value from multi-line systemctl show output.
func parseProperty(output, key string) string {
	prefix := key + "="
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimPrefix(line, prefix)
		}
	}
	return ""
}
