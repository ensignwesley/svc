package manifest_test

import (
	"fmt"
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

// ── Adversarial / edge-case tests ────────────────────────────────────────────

func TestValidateNegativePort(t *testing.T) {
	m := &manifest.Manifest{
		Meta: manifest.Meta{Version: 1},
		Services: map[string]manifest.Service{
			"svc": {Description: "test", Port: -1},
		},
	}
	result := manifest.Validate(m)
	if result.Valid() {
		t.Fatal("expected invalid for negative port")
	}
	found := false
	for _, e := range result.Errors {
		if strings.Contains(e, "out of range") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected out-of-range error, got: %v", result.Errors)
	}
}

func TestValidatePortTooHigh(t *testing.T) {
	m := &manifest.Manifest{
		Meta: manifest.Meta{Version: 1},
		Services: map[string]manifest.Service{
			"svc": {Description: "test", Port: 99999},
		},
	}
	result := manifest.Validate(m)
	if result.Valid() {
		t.Fatal("expected invalid for port > 65535")
	}
	found := false
	for _, e := range result.Errors {
		if strings.Contains(e, "out of range") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected out-of-range error, got: %v", result.Errors)
	}
}

func TestValidatePortBoundaries(t *testing.T) {
	// Port 1 and 65535 must be valid.
	for _, port := range []int{1, 1024, 8080, 65535} {
		m := &manifest.Manifest{
			Meta: manifest.Meta{Version: 1},
			Services: map[string]manifest.Service{
				"svc": {Description: "test", Port: port},
			},
		}
		result := manifest.Validate(m)
		if !result.Valid() {
			t.Errorf("port %d should be valid, got errors: %v", port, result.Errors)
		}
	}

	// Port 0 is only valid when health_url is set.
	m := &manifest.Manifest{
		Meta: manifest.Meta{Version: 1},
		Services: map[string]manifest.Service{
			"svc": {Description: "test", Port: 0, HealthURL: "https://example.com/health"},
		},
	}
	result := manifest.Validate(m)
	if !result.Valid() {
		t.Errorf("port 0 with health_url should be valid, got: %v", result.Errors)
	}
}

func TestValidateEmptyServiceID(t *testing.T) {
	m := &manifest.Manifest{
		Meta: manifest.Meta{Version: 1},
		Services: map[string]manifest.Service{
			"": {Description: "empty id", Port: 8080},
		},
	}
	result := manifest.Validate(m)
	if result.Valid() {
		t.Fatal("expected invalid for empty service ID")
	}
	found := false
	for _, e := range result.Errors {
		if strings.Contains(e, "empty") || strings.Contains(e, "ID") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected empty-ID error, got: %v", result.Errors)
	}
}

func TestValidateHealthURLBadScheme(t *testing.T) {
	cases := []struct {
		name string
		url  string
	}{
		{"no scheme", "localhost:8080/health"},
		{"garbage", "not a url at all!!!"},
		{"ftp scheme", "ftp://example.com/health"},
		{"file scheme", "file:///etc/passwd"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := &manifest.Manifest{
				Meta: manifest.Meta{Version: 1},
				Services: map[string]manifest.Service{
					"svc": {Description: "test", HealthURL: tc.url},
				},
			}
			result := manifest.Validate(m)
			if result.Valid() {
				t.Errorf("expected invalid for health_url %q", tc.url)
			}
		})
	}
}

func TestValidateHealthURLValid(t *testing.T) {
	cases := []string{
		"http://localhost:8080/health",
		"https://example.com/health",
		"http://192.168.1.1:9090/healthz",
		"https://my-service.internal/",
	}
	for _, u := range cases {
		t.Run(u, func(t *testing.T) {
			m := &manifest.Manifest{
				Meta: manifest.Meta{Version: 1},
				Services: map[string]manifest.Service{
					"svc": {Description: "test", HealthURL: u},
				},
			}
			result := manifest.Validate(m)
			if !result.Valid() {
				t.Errorf("expected valid for health_url %q, got: %v", u, result.Errors)
			}
		})
	}
}

func TestLoadEmptyFile(t *testing.T) {
	tmp := writeTemp(t, "")
	_, err := manifest.Load(tmp)
	if err == nil {
		t.Fatal("expected error for empty file")
	}
	if !strings.Contains(err.Error(), "version") {
		t.Errorf("expected version error, got: %v", err)
	}
}

func TestLoadGarbageYAML(t *testing.T) {
	tmp := writeTemp(t, "not: yaml: [[[")
	_, err := manifest.Load(tmp)
	if err == nil {
		t.Fatal("expected error for garbage YAML")
	}
}

func TestLoadBinaryContent(t *testing.T) {
	f := filepath.Join(t.TempDir(), "binary.yaml")
	if err := os.WriteFile(f, []byte{0x00, 0x01, 0x02, 0xff, 0xfe}, 0644); err != nil {
		t.Fatal(err)
	}
	_, err := manifest.Load(f)
	if err == nil {
		t.Fatal("expected error for binary file")
	}
}

func TestValidateDuplicateKeysRejectedByYAML(t *testing.T) {
	yaml := `
manifest:
  version: 1
services:
  svc-a:
    port: 8080
    description: "First"
  svc-a:
    port: 9090
    description: "Duplicate key"
`
	_, err := manifest.ParseManifest([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for duplicate service keys")
	}
}

func TestValidateMassiveManifest(t *testing.T) {
	// 1000 services should parse and validate cleanly without panic or timeout.
	services := make(map[string]manifest.Service, 1000)
	for i := 0; i < 1000; i++ {
		services[fmt.Sprintf("svc-%04d", i)] = manifest.Service{
			Description: fmt.Sprintf("Service %d", i),
			Port:        3000 + i,
		}
	}
	m := &manifest.Manifest{
		Meta:     manifest.Meta{Version: 1, Host: "localhost"},
		Services: services,
	}
	result := manifest.Validate(m)
	if !result.Valid() {
		t.Fatalf("expected valid massive manifest, got errors: %v", result.Errors)
	}
}

// ── LoadDir / LoadAuto tests ─────────────────────────────────────────────────

func TestLoadDirMergesTwoFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "web.yaml"), `
manifest:
  version: 1
  host: localhost
services:
  blog:
    description: "Blog"
    health_url: "https://example.com/"
`)
	writeFile(t, filepath.Join(dir, "infra.yaml"), `
manifest:
  version: 1
services:
  db:
    description: "Database"
    port: 5432
`)

	m, err := manifest.LoadDir(dir)
	if err != nil {
		t.Fatalf("LoadDir() error: %v", err)
	}
	if len(m.Services) != 2 {
		t.Fatalf("expected 2 services, got %d", len(m.Services))
	}
	if _, ok := m.Services["blog"]; !ok {
		t.Error("expected service 'blog'")
	}
	if _, ok := m.Services["db"]; !ok {
		t.Error("expected service 'db'")
	}
}

func TestLoadDirUsesFirstFileMeta(t *testing.T) {
	dir := t.TempDir()
	// Files are sorted alphabetically; a.yaml comes first.
	writeFile(t, filepath.Join(dir, "a.yaml"), `
manifest:
  version: 1
  host: primary-host
services:
  alpha:
    description: "Alpha"
    port: 8001
`)
	writeFile(t, filepath.Join(dir, "b.yaml"), `
manifest:
  version: 1
  host: other-host
services:
  beta:
    description: "Beta"
    port: 8002
`)

	m, err := manifest.LoadDir(dir)
	if err != nil {
		t.Fatalf("LoadDir() error: %v", err)
	}
	if m.Meta.Host != "primary-host" {
		t.Errorf("expected host from first file 'primary-host', got %q", m.Meta.Host)
	}
}

func TestLoadDirRejectsDuplicateServiceIDs(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.yaml"), `
manifest:
  version: 1
services:
  shared-svc:
    description: "First definition"
    port: 8080
`)
	writeFile(t, filepath.Join(dir, "b.yaml"), `
manifest:
  version: 1
services:
  shared-svc:
    description: "Duplicate"
    port: 9090
`)

	_, err := manifest.LoadDir(dir)
	if err == nil {
		t.Fatal("expected error for duplicate service ID")
	}
	if !strings.Contains(err.Error(), "duplicate") {
		t.Errorf("expected 'duplicate' in error, got: %v", err)
	}
}

func TestLoadDirEmptyDirectory(t *testing.T) {
	dir := t.TempDir()
	_, err := manifest.LoadDir(dir)
	if err == nil {
		t.Fatal("expected error for empty directory")
	}
	if !strings.Contains(err.Error(), "no *.yaml") {
		t.Errorf("expected 'no *.yaml' in error, got: %v", err)
	}
}

func TestLoadDirNonexistentDirectory(t *testing.T) {
	_, err := manifest.LoadDir("/nonexistent/path/xyz")
	if err == nil {
		t.Fatal("expected error for nonexistent directory")
	}
}

func TestLoadAutoFile(t *testing.T) {
	tmp := writeTemp(t, `
manifest:
  version: 1
services:
  svc:
    description: "test"
    port: 8080
`)
	m, err := manifest.LoadAuto(tmp)
	if err != nil {
		t.Fatalf("LoadAuto() on file error: %v", err)
	}
	if len(m.Services) != 1 {
		t.Errorf("expected 1 service, got %d", len(m.Services))
	}
}

func TestLoadAutoDirectory(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "services.yaml"), `
manifest:
  version: 1
services:
  app:
    description: "App"
    port: 3000
`)

	m, err := manifest.LoadAuto(dir)
	if err != nil {
		t.Fatalf("LoadAuto() on directory error: %v", err)
	}
	if len(m.Services) != 1 {
		t.Errorf("expected 1 service, got %d", len(m.Services))
	}
}

func TestLoadAutoNonexistent(t *testing.T) {
	_, err := manifest.LoadAuto("/nonexistent/services.yaml")
	if err == nil {
		t.Fatal("expected error for nonexistent path")
	}
	if !strings.Contains(err.Error(), "svc init") {
		t.Errorf("expected 'svc init' hint in error, got: %v", err)
	}
}

func TestLoadDirIgnoresNonYAMLFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "services.yaml"), `
manifest:
  version: 1
services:
  app:
    description: "App"
    port: 3000
`)
	// These should be ignored:
	writeFile(t, filepath.Join(dir, "README.md"), "# readme")
	writeFile(t, filepath.Join(dir, ".gitignore"), "*.bak")
	writeFile(t, filepath.Join(dir, "notes.txt"), "some notes")

	m, err := manifest.LoadDir(dir)
	if err != nil {
		t.Fatalf("LoadDir() error: %v", err)
	}
	if len(m.Services) != 1 {
		t.Errorf("expected 1 service (non-yaml files ignored), got %d", len(m.Services))
	}
}

func TestLoadDirRejectsWrongVersion(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "bad.yaml"), `
manifest:
  version: 2
services:
  svc:
    port: 8080
`)
	_, err := manifest.LoadDir(dir)
	if err == nil {
		t.Fatal("expected error for wrong manifest version")
	}
	if !strings.Contains(err.Error(), "version") {
		t.Errorf("expected 'version' in error, got: %v", err)
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
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
