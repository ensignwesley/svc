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

func TestValidateValid(t *testing.T) {
	m := &manifest.Manifest{
		Meta: manifest.Meta{Version: 1, Host: "localhost"},
		Services: map[string]manifest.Service{
			"svc": {Description: "test service", Port: 8080},
		},
	}
	result := manifest.Validate(m)
	if !result.Valid() {
		t.Fatalf("expected valid, got errors: %v", result.Errors)
	}
	if len(result.Warnings) != 0 {
		t.Errorf("expected no warnings, got: %v", result.Warnings)
	}
}

func TestValidateErrorMissingVersion(t *testing.T) {
	m := &manifest.Manifest{
		Meta: manifest.Meta{Host: "localhost"},
		Services: map[string]manifest.Service{
			"svc": {Port: 8080},
		},
	}
	result := manifest.Validate(m)
	if result.Valid() {
		t.Fatal("expected invalid for missing version")
	}
	if !strings.Contains(result.Errors[0], "version") {
		t.Errorf("expected version error, got: %v", result.Errors[0])
	}
}

func TestValidateErrorMissingPortAndURL(t *testing.T) {
	m := &manifest.Manifest{
		Meta: manifest.Meta{Version: 1},
		Services: map[string]manifest.Service{
			"svc": {Description: "no port"},
		},
	}
	result := manifest.Validate(m)
	if result.Valid() {
		t.Fatal("expected invalid for missing port and health_url")
	}
	if !strings.Contains(result.Errors[0], "port") {
		t.Errorf("expected port error, got: %v", result.Errors[0])
	}
}

func TestValidateWarningRepoWithoutVersion(t *testing.T) {
	m := &manifest.Manifest{
		Meta: manifest.Meta{Version: 1},
		Services: map[string]manifest.Service{
			"svc": {Description: "test", Port: 8080, Repo: "owner/repo"},
		},
	}
	result := manifest.Validate(m)
	if !result.Valid() {
		t.Fatalf("expected valid (warnings not errors), got: %v", result.Errors)
	}
	if len(result.Warnings) == 0 {
		t.Fatal("expected warning for repo without version")
	}
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "version") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected version warning, got: %v", result.Warnings)
	}
}

func TestValidateWarningEmptyDescription(t *testing.T) {
	m := &manifest.Manifest{
		Meta: manifest.Meta{Version: 1},
		Services: map[string]manifest.Service{
			"svc": {Port: 8080},
		},
	}
	result := manifest.Validate(m)
	if !result.Valid() {
		t.Fatalf("expected valid, got: %v", result.Errors)
	}
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "description") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected description warning, got: %v", result.Warnings)
	}
}

func TestValidateMultipleErrors(t *testing.T) {
	m := &manifest.Manifest{
		Meta: manifest.Meta{Version: 1},
		Services: map[string]manifest.Service{
			"alpha": {Description: "no port or url"},
			"beta":  {Description: "also no port or url"},
		},
	}
	result := manifest.Validate(m)
	if result.Valid() {
		t.Fatal("expected invalid")
	}
	if len(result.Errors) != 2 {
		t.Errorf("expected 2 errors, got %d: %v", len(result.Errors), result.Errors)
	}
}

func TestParseManifest(t *testing.T) {
	yaml := `
manifest:
  version: 1
  host: testhost
services:
  blog:
    description: "Test blog"
    health_url: "https://example.com/"
`
	m, err := manifest.ParseManifest([]byte(yaml))
	if err != nil {
		t.Fatalf("ParseManifest() error: %v", err)
	}
	if m.Meta.Host != "testhost" {
		t.Errorf("expected host testhost, got %q", m.Meta.Host)
	}
	if len(m.Services) != 1 {
		t.Errorf("expected 1 service, got %d", len(m.Services))
	}
}

func TestParseManifestInvalidYAML(t *testing.T) {
	// Indentation error that triggers yaml parse failure
	_, err := manifest.ParseManifest([]byte("manifest:\n  version: 1\n services:\n  foo: bar"))
	if err == nil {
		t.Fatal("expected error for invalid YAML")
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
