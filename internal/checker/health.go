package checker

import (
	"fmt"
	"net/http"
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
		result.Err = summariseError(err)
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

// summariseError converts a net/http error into an operator-readable message.
func summariseError(err error) string {
	msg := err.Error()
	// connection refused
	if contains(msg, "connection refused") {
		return "connection refused"
	}
	// timeout
	if contains(msg, "timeout") || contains(msg, "deadline exceeded") {
		return "timeout"
	}
	// DNS
	if contains(msg, "no such host") || contains(msg, "dial tcp") {
		return "DNS/connection error"
	}
	return msg
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
