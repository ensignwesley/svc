package checker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// VersionResult holds the result of a GitHub version check.
type VersionResult struct {
	Repo     string
	Current  string
	Latest   string
	Status   string // "current", "behind", "ahead", "unknown"
	Err      error
}

var versionClient = &http.Client{Timeout: 8 * time.Second}

// CheckVersion compares current vs latest GitHub release for a repo.
func CheckVersion(repo, current string, maxMajor int) VersionResult {
	if repo == "" || current == "" {
		return VersionResult{Repo: repo, Current: current, Status: "unknown"}
	}

	latest, err := fetchLatestRelease(repo)
	if err != nil {
		return VersionResult{Repo: repo, Current: current, Err: err, Status: "unknown"}
	}

	// Strip leading 'v' for comparison
	cur := strings.TrimPrefix(current, "v")
	lat := strings.TrimPrefix(latest, "v")

	if maxMajor > 0 {
		latMaj := majorVersion(lat)
		if latMaj > maxMajor {
			// Latest exceeds constraint — skip
			return VersionResult{Repo: repo, Current: current, Latest: latest, Status: "current"}
		}
	}

	if cur == lat {
		return VersionResult{Repo: repo, Current: current, Latest: latest, Status: "current"}
	}
	if semverLess(cur, lat) {
		return VersionResult{Repo: repo, Current: current, Latest: latest, Status: "behind"}
	}
	return VersionResult{Repo: repo, Current: current, Latest: latest, Status: "ahead"}
}

func fetchLatestRelease(repo string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	req.Header.Set("User-Agent", "svc/0.1")
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := versionClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("GitHub API returned %d for %s", resp.StatusCode, repo)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}
	return release.TagName, nil
}

func majorVersion(v string) int {
	parts := strings.Split(v, ".")
	if len(parts) == 0 {
		return 0
	}
	n, _ := strconv.Atoi(parts[0])
	return n
}

func semverLess(a, b string) bool {
	aParts := parseSemver(a)
	bParts := parseSemver(b)
	for i := 0; i < 3; i++ {
		if aParts[i] < bParts[i] {
			return true
		}
		if aParts[i] > bParts[i] {
			return false
		}
	}
	return false
}

func parseSemver(v string) [3]int {
	parts := strings.Split(v, ".")
	var result [3]int
	for i := 0; i < 3 && i < len(parts); i++ {
		// Strip any non-numeric suffix (e.g. "1" from "1-alpine")
		n := ""
		for _, c := range parts[i] {
			if c >= '0' && c <= '9' {
				n += string(c)
			} else {
				break
			}
		}
		result[i], _ = strconv.Atoi(n)
	}
	return result
}
