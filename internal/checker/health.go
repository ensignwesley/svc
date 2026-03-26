package checker

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// HealthResult holds the result of a single health check.
type HealthResult struct {
	ServiceID  string
	URL        string
	Up         bool
	StatusCode int
	LatencyMS  int64
	Err        string
}

// CheckHealth polls a single health endpoint and returns the result.
func CheckHealth(id, url string, timeoutSec int) HealthResult {
	client := &http.Client{
		Timeout: time.Duration(timeoutSec) * time.Second,
		// Disable keep-alives so each check opens a fresh connection.
		// Avoids false "up" results from reused connections to dead services.
		Transport: &http.Transport{DisableKeepAlives: true},
	}

	start := time.Now()
	resp, err := client.Get(url)
	elapsed := time.Since(start).Milliseconds()

	result := HealthResult{
		ServiceID: id,
		URL:       url,
		LatencyMS: elapsed,
	}

	if err != nil {
		result.Up = false
		result.Err = summariseError(err, timeoutSec)
		return result
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode
	result.Up = resp.StatusCode == 200
	if !result.Up {
		result.Err = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}
	return result
}

// CheckAllHealth runs health checks concurrently for all provided (id, url) pairs.
func CheckAllHealth(targets map[string]string, timeoutSec int) []HealthResult {
	results := make([]HealthResult, 0, len(targets))
	var mu sync.Mutex
	var wg sync.WaitGroup

	for id, url := range targets {
		wg.Add(1)
		go func(id, url string) {
			defer wg.Done()
			r := CheckHealth(id, url, timeoutSec)
			mu.Lock()
			results = append(results, r)
			mu.Unlock()
		}(id, url)
	}

	wg.Wait()
	return results
}

// summariseError converts a net/http error into an actionable operator message.
// Error messages should tell the operator what happened AND what to do about it.
func summariseError(err error, timeoutSec int) string {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "connection refused"):
		return "connection refused"
	case strings.Contains(msg, "timeout") || strings.Contains(msg, "deadline exceeded"):
		return fmt.Sprintf("timeout after %ds (--timeout to increase)", timeoutSec)
	case strings.Contains(msg, "no such host"):
		return "DNS lookup failed — check health_url hostname"
	case strings.Contains(msg, "dial tcp"):
		return "host unreachable"
	case strings.Contains(msg, "certificate"):
		return "TLS certificate error (--no-verify to skip)"
	default:
		return msg
	}
}
