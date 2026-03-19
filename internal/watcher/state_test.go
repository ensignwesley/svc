package watcher_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ensignwesley/svc/internal/watcher"
)

func TestNewWatchState(t *testing.T) {
	ws := watcher.NewWatchState()
	if ws.Services == nil {
		t.Fatal("expected non-nil Services map")
	}
}

func TestGetOrInit(t *testing.T) {
	ws := watcher.NewWatchState()
	ss := ws.GetOrInit("dead-drop")
	if ss == nil {
		t.Fatal("expected non-nil service state")
	}
	if ss.Status != watcher.StatusUnknown {
		t.Errorf("expected StatusUnknown, got %v", ss.Status)
	}
	// Second call returns same pointer.
	ss2 := ws.GetOrInit("dead-drop")
	if ss != ss2 {
		t.Error("expected same pointer on second GetOrInit")
	}
}

func TestSaveAndLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "watch-state.json")

	ws := watcher.NewWatchState()
	ss := ws.GetOrInit("blog")
	ss.Status = watcher.StatusUp
	ss.ConsecutiveFailures = 0
	ss.LastCheck = time.Now().UTC()

	if err := watcher.Save(path, ws); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := watcher.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	blog, ok := loaded.Services["blog"]
	if !ok {
		t.Fatal("expected 'blog' in loaded state")
	}
	if blog.Status != watcher.StatusUp {
		t.Errorf("expected StatusUp, got %v", blog.Status)
	}
}

func TestLoadMissingFile(t *testing.T) {
	ws, err := watcher.Load("/nonexistent/path/state.json")
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if ws.Services == nil {
		t.Fatal("expected empty state, not nil")
	}
}

func TestLoadCorruptFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	os.WriteFile(path, []byte("not json {{{"), 0644)
	ws, err := watcher.Load(path)
	if err != nil {
		t.Fatalf("expected graceful recovery from corrupt file, got: %v", err)
	}
	if ws.Services == nil {
		t.Fatal("expected empty state after corrupt file")
	}
}

func TestStatusString(t *testing.T) {
	cases := []struct {
		s    watcher.ServiceStatus
		want string
	}{
		{watcher.StatusUnknown, "unknown"},
		{watcher.StatusUp, "up"},
		{watcher.StatusDegraded, "degraded"},
		{watcher.StatusDown, "down"},
	}
	for _, c := range cases {
		if got := c.s.String(); got != c.want {
			t.Errorf("Status(%d).String() = %q, want %q", c.s, got, c.want)
		}
	}
}
