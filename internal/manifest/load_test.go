package manifest_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ensignwesley/svc/internal/manifest"
)

func TestLoadValid(t *testing.T) {
	m, err := manifest.Load("../../testdata/services.yaml")
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if m.Meta.Version != 1 {
		t.Errorf("expected version 1, got %d", m.Meta.Version)
	}
	if m.Meta.Host != "localhost" {
		t.Errorf("expected host localhost, got %q", m.Meta.Host)
	}
	if len(m.Services) != 2 {
		t.Errorf("expected 2 services, got %d", len(m.Services))
	}

	dd, ok := m.Services["dead-drop"]
	if !ok {
		t.Fatal("expected service 'dead-drop'")
	}
	if dd.Port != 3001 {
		t.Errorf("expected dead-drop port 3001, got %d", dd.Port)
	}
	if dd.SystemdUnit != "dead-drop.service" {
		t.Errorf("expected systemd unit 'dead-drop.service', got %q", dd.SystemdUnit)
	}

	blog, ok := m.Services["blog"]
	if !ok {
		t.Fatal("expected service 'blog'")
	}
	if blog.HealthURL != "https://wesley.thesisko.com/" {
		t.Errorf("unexpected blog health_url: %q", blog.HealthURL)
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := manifest.Load("nonexistent.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !strings.Contains(err.Error(), "svc init") {
		t.Errorf("error message should mention 'svc init', got: %v", err)
	}
}

func TestLoadMissingVersion(t *testing.T) {
	yaml := `
manifest:
  host: localhost
services:
  svc:
    port: 8080
`
	tmp := writeTemp(t, yaml)
	_, err := manifest.Load(tmp)
	if err == nil {
		t.Fatal("expected error for missing version")
	}
}

func TestLoadMissingPortAndURL(t *testing.T) {
	yaml := `
manifest:
  version: 1
services:
  svc:
    description: "no port, no url"
`
	tmp := writeTemp(t, yaml)
	_, err := manifest.Load(tmp)
	if err == nil {
		t.Fatal("expected error for missing port and health_url")
	}
	if !strings.Contains(err.Error(), "port") {
		t.Errorf("error should mention 'port', got: %v", err)
	}
}

func TestResolveHealthURL(t *testing.T) {
	m := &manifest.Manifest{
		Meta: manifest.Meta{Host: "localhost"},
	}

	// Port only — derives URL
	svc := manifest.Service{Port: 3001}
	url := manifest.ResolveHealthURL(m, svc)
	if url != "http://localhost:3001/health" {
		t.Errorf("expected derived URL, got %q", url)
	}

	// Explicit health_url overrides
	svc2 := manifest.Service{Port: 3001, HealthURL: "http://localhost:3001/healthz"}
	url2 := manifest.ResolveHealthURL(m, svc2)
	if url2 != "http://localhost:3001/healthz" {
		t.Errorf("expected explicit URL, got %q", url2)
	}
}

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	f := filepath.Join(t.TempDir(), "services.yaml")
	if err := os.WriteFile(f, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return f
}
