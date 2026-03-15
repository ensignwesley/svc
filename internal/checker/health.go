package checker

import (
	"context"
	"net/http"
	"sync"
	"time"
)

// HealthResult holds the result of a single health check.
type HealthResult struct {
	ID        string
	URL       string
	Up        bool
	LatencyMs int64
	Err       error
}

// CheckAll polls all given URLs concurrently and returns results.
func CheckAll(targets map[string]string, timeoutSecs int) []HealthResult {
	if timeoutSecs <= 0 {
		timeoutSecs = 5
	}
	client := &http.Client{
		Timeout: time.Duration(timeoutSecs) * time.Second,
	}

	results := make([]HealthResult, 0, len(targets))
	var mu sync.Mutex
	var wg sync.WaitGroup

	for id, url := range targets {
		wg.Add(1)
		go func(id, url string) {
			defer wg.Done()
			r := checkOne(client, id, url)
			mu.Lock()
			results = append(results, r)
			mu.Unlock()
		}(id, url)
	}

	wg.Wait()
	return results
}

func checkOne(client *http.Client, id, url string) HealthResult {
	start := time.Now()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return HealthResult{ID: id, URL: url, Up: false, Err: err}
	}
	req.Header.Set("User-Agent", "svc/0.1")

	resp, err := client.Do(req)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return HealthResult{ID: id, URL: url, Up: false, LatencyMs: latency, Err: err}
	}
	resp.Body.Close()

	up := resp.StatusCode >= 200 && resp.StatusCode < 300
	return HealthResult{ID: id, URL: url, Up: up, LatencyMs: latency}
}
