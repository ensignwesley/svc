package watcher_test

// Tests for hot-reload behaviour: manifest is re-read on every tick so edits
// take effect without restarting the process or resetting alert state.

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ensignwesley/svc/internal/watcher"
)

// writeManifest writes a services.yaml to dir with the given services block.
func writeManifest(t *testing.T, path, services string) {
	t.Helper()
	content := "manifest:\n  version: 1\n  host: localhost\nservices:\n" + services
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writing manifest: %v", err)
	}
}

func TestRunCheckOnce_BadManifest_SkipsTick(t *testing.T) {
	// If the manifest is unreadable, runCheck should log the error and return
	// without modifying state. Alert counts and alerted flags must survive.
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "services.yaml")

	// Write garbage.
	if err := os.WriteFile(manifestPath, []byte("not: yaml: [[["), 0644); err != nil {
		t.Fatal(err)
	}

	state := watcher.NewWatchState()
	// Pre-seed a service with a non-zero failure count to prove state is untouched.
	ss := state.GetOrInit("sentinel")
	ss.Status = watcher.StatusDegraded
	ss.ConsecutiveFailures = 2

	cfg := watcher.Config{
		ManifestPath: manifestPath,
		StatePath:    filepath.Join(dir, "state.json"),
		FailThreshold: 3,
		TimeoutSec:   1,
		NoSystemd:    true,
	}

	var buf bytes.Buffer
	watcher.RunCheckOnce(cfg, state, &buf)

	// State must be unchanged.
	if ss.ConsecutiveFailures != 2 {
		t.Errorf("expected ConsecutiveFailures=2 after bad reload, got %d", ss.ConsecutiveFailures)
	}
	if ss.Status != watcher.StatusDegraded {
		t.Errorf("expected StatusDegraded after bad reload, got %v", ss.Status)
	}

	// Output must mention the reload failure.
	if !strings.Contains(buf.String(), "manifest reload failed") {
		t.Errorf("expected reload failure message in output, got: %q", buf.String())
	}
}

func TestRunCheckOnce_BadManifest_AlertedFlagSurvives(t *testing.T) {
	// Alerted flag must not be cleared by a failed reload — a mis-saved manifest
	// edit must not suppress an in-flight alert.
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "services.yaml")
	os.WriteFile(manifestPath, []byte("{invalid"), 0644)

	state := watcher.NewWatchState()
	ss := state.GetOrInit("web")
	ss.Status = watcher.StatusDown
	ss.Alerted = true
	ss.ConsecutiveFailures = 5

	cfg := watcher.Config{
		ManifestPath:  manifestPath,
		StatePath:     filepath.Join(dir, "state.json"),
		FailThreshold: 3,
		TimeoutSec:    1,
		NoSystemd:     true,
	}

	var buf bytes.Buffer
	watcher.RunCheckOnce(cfg, state, &buf)

	if !ss.Alerted {
		t.Error("Alerted flag must survive a failed manifest reload")
	}
	if ss.ConsecutiveFailures != 5 {
		t.Errorf("ConsecutiveFailures must survive a failed reload, got %d", ss.ConsecutiveFailures)
	}
}

func TestRunCheckOnce_MissingManifest_SkipsTick(t *testing.T) {
	// Nonexistent manifest file: same graceful-skip behaviour.
	dir := t.TempDir()

	state := watcher.NewWatchState()
	ss := state.GetOrInit("app")
	ss.ConsecutiveFailures = 1

	cfg := watcher.Config{
		ManifestPath:  filepath.Join(dir, "nonexistent.yaml"),
		StatePath:     filepath.Join(dir, "state.json"),
		FailThreshold: 3,
		TimeoutSec:    1,
		NoSystemd:     true,
	}

	var buf bytes.Buffer
	watcher.RunCheckOnce(cfg, state, &buf)

	if ss.ConsecutiveFailures != 1 {
		t.Errorf("state must be untouched for missing manifest, got failures=%d", ss.ConsecutiveFailures)
	}
	if !strings.Contains(buf.String(), "manifest reload failed") {
		t.Errorf("expected reload failure message, got: %q", buf.String())
	}
}

func TestRunCheckOnce_NewServiceAppearsAfterReload(t *testing.T) {
	// After a manifest edit that adds a service, the next tick should initialise
	// that service's state from scratch — without restarting the loop.
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "services.yaml")

	// Start with one service.
	writeManifest(t, manifestPath,
		"  alpha:\n    description: \"Alpha\"\n    health_url: \"http://localhost:19991/health\"\n")

	state := watcher.NewWatchState()
	cfg := watcher.Config{
		ManifestPath:  manifestPath,
		StatePath:     filepath.Join(dir, "state.json"),
		FailThreshold: 3,
		TimeoutSec:    1,
		NoSystemd:     true,
	}

	var buf bytes.Buffer
	watcher.RunCheckOnce(cfg, state, &buf)

	// alpha should now be in state.
	if _, ok := state.Services["alpha"]; !ok {
		t.Fatal("expected 'alpha' in state after first tick")
	}

	// Now edit manifest to add beta.
	writeManifest(t, manifestPath,
		"  alpha:\n    description: \"Alpha\"\n    health_url: \"http://localhost:19991/health\"\n"+
			"  beta:\n    description: \"Beta\"\n    health_url: \"http://localhost:19992/health\"\n")

	buf.Reset()
	watcher.RunCheckOnce(cfg, state, &buf)

	// beta should now be in state — picked up by reload, no restart needed.
	if _, ok := state.Services["beta"]; !ok {
		t.Fatal("expected 'beta' in state after manifest edit — hot-reload failed")
	}
}

func TestRunCheckOnce_AlertStatePreservedAcrossGoodReload(t *testing.T) {
	// A successful manifest reload (valid file, service still present) must not
	// reset ConsecutiveFailures or the Alerted flag for services that were
	// already in a degraded/down state.
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "services.yaml")
	writeManifest(t, manifestPath,
		"  app:\n    description: \"App\"\n    health_url: \"http://localhost:19993/health\"\n")

	state := watcher.NewWatchState()
	// Pre-seed degraded state.
	ss := state.GetOrInit("app")
	ss.Status = watcher.StatusDegraded
	ss.ConsecutiveFailures = 2
	ss.Alerted = false

	cfg := watcher.Config{
		ManifestPath:  manifestPath,
		StatePath:     filepath.Join(dir, "state.json"),
		FailThreshold: 5, // high threshold so we don't alert on this tick
		TimeoutSec:    1,
		NoSystemd:     true,
	}

	var buf bytes.Buffer
	watcher.RunCheckOnce(cfg, state, &buf)

	// Service is unreachable (nothing listening on 19993), so consecutive
	// failures should increment — not reset.
	appState := state.Services["app"]
	if appState.ConsecutiveFailures < 2 {
		t.Errorf("ConsecutiveFailures should not reset across reload, got %d", appState.ConsecutiveFailures)
	}
}
