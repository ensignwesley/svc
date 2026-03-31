package manifest

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// ValidationResult holds errors and warnings from manifest validation.
// Errors block valid usage; warnings are advisory.
type ValidationResult struct {
	Errors   []string
	Warnings []string
}

// Valid returns true if there are no errors (warnings do not block validity).
func (v *ValidationResult) Valid() bool {
	return len(v.Errors) == 0
}

// Load reads and validates a manifest file from the given path.
func Load(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("services.yaml not found at %q\nRun 'svc init' to create one, or use --file to specify a path", path)
		}
		return nil, fmt.Errorf("reading manifest: %w", err)
	}

	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing manifest: %w", err)
	}

	vr := Validate(&m)
	if !vr.Valid() {
		return nil, fmt.Errorf("%s", vr.Errors[0])
	}

	// Apply defaults.
	if m.Meta.Host == "" {
		m.Meta.Host = "localhost"
	}

	return &m, nil
}

// LoadAuto loads a manifest from a file path or a directory.
// If path is a directory, it calls LoadDir and merges all *.yaml files found.
// If path is a file, it calls Load directly.
func LoadAuto(path string) (*Manifest, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("services.yaml not found at %q\nRun 'svc init' to create one, or use --file to specify a path", path)
		}
		return nil, fmt.Errorf("stat %q: %w", path, err)
	}
	if info.IsDir() {
		return LoadDir(path)
	}
	return Load(path)
}

// LoadDir merges all *.yaml files in dir into a single Manifest.
// Service IDs must be unique across files — duplicates are an error.
// The manifest.version and manifest.host from the first file encountered
// (alphabetically) are used as the merged Meta; per-file meta mismatches
// are ignored (only version must be 1 in every file).
func LoadDir(dir string) (*Manifest, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading directory %q: %w", dir, err)
	}

	var yamlFiles []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") {
			yamlFiles = append(yamlFiles, filepath.Join(dir, name))
		}
	}

	if len(yamlFiles) == 0 {
		return nil, fmt.Errorf("no *.yaml files found in directory %q", dir)
	}

	// Sort for deterministic merge order.
	sort.Strings(yamlFiles)

	merged := &Manifest{
		Services: make(map[string]Service),
	}
	// serviceOrigin tracks which file each service ID came from, for duplicate diagnostics.
	serviceOrigin := make(map[string]string)
	metaSet := false

	for _, path := range yamlFiles {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading %q: %w", path, err)
		}

		var m Manifest
		if err := yaml.Unmarshal(data, &m); err != nil {
			return nil, fmt.Errorf("parsing %q: %w", path, err)
		}

		// Require version 1 in every file.
		if m.Meta.Version != 1 {
			return nil, fmt.Errorf("%q: manifest.version must be 1 (got %d)", path, m.Meta.Version)
		}

		// Use meta from the first file.
		if !metaSet {
			merged.Meta = m.Meta
			metaSet = true
		}

		// Merge services — reject duplicates, naming both files.
		for id, svc := range m.Services {
			if first, exists := serviceOrigin[id]; exists {
				return nil, fmt.Errorf("duplicate service ID %q: defined in both %q and %q", id, first, path)
			}
			merged.Services[id] = svc
			serviceOrigin[id] = path
		}
	}

	// Validate the merged manifest.
	vr := Validate(merged)
	if !vr.Valid() {
		return nil, fmt.Errorf("%s", vr.Errors[0])
	}

	// Apply defaults.
	if merged.Meta.Host == "" {
		merged.Meta.Host = "localhost"
	}

	return merged, nil
}

// ParseManifest parses raw YAML bytes into a Manifest without validating.
// Use Validate() on the result to check semantics.
func ParseManifest(data []byte) (*Manifest, error) {
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing manifest: %w", err)
	}
	return &m, nil
}

// Validate checks manifest semantics and returns errors and warnings.
// It does not make any network calls. Safe to call from CI.
func Validate(m *Manifest) *ValidationResult {
	result := &ValidationResult{}

	// Version check.
	if m.Meta.Version == 0 {
		result.Errors = append(result.Errors, "manifest.version is required (set to 1)")
	} else if m.Meta.Version != 1 {
		result.Errors = append(result.Errors, fmt.Sprintf("unsupported manifest version %d (expected 1)", m.Meta.Version))
	}

	// Per-service checks. Sort IDs for deterministic output.
	ids := make([]string, 0, len(m.Services))
	for id := range m.Services {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	for _, id := range ids {
		svc := m.Services[id]

		// Empty service ID is invalid.
		if strings.TrimSpace(id) == "" {
			result.Errors = append(result.Errors,
				"service ID must not be empty or whitespace-only")
		}

		if svc.Port == 0 && svc.HealthURL == "" {
			result.Errors = append(result.Errors,
				fmt.Sprintf("service %q: one of 'port' or 'health_url' is required", id))
		}

		// Port range validation (only when port is the health mechanism).
		if svc.Port != 0 {
			if svc.Port < 1 || svc.Port > 65535 {
				result.Errors = append(result.Errors,
					fmt.Sprintf("service %q: port %d is out of range (1–65535)", id, svc.Port))
			}
		}

		// health_url must be a valid absolute URL if provided.
		if svc.HealthURL != "" {
			if u, err := url.Parse(svc.HealthURL); err != nil || (u.Scheme != "http" && u.Scheme != "https") {
				result.Errors = append(result.Errors,
					fmt.Sprintf("service %q: health_url must be an absolute http/https URL (got %q)", id, svc.HealthURL))
			}
		}

		if svc.Repo != "" && svc.Version == "" {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("service %q: repo is set without version (version drift check will be skipped)", id))
		}
		if svc.Description == "" {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("service %q: description is empty", id))
		}
	}

	return result
}



// ResolveHealthURL returns the effective health URL for a service.
// Explicit health_url takes precedence; otherwise derives from host+port.
func ResolveHealthURL(m *Manifest, svc Service) string {
	if svc.HealthURL != "" {
		return svc.HealthURL
	}
	host := m.Meta.Host
	if host == "" {
		host = "localhost"
	}
	return fmt.Sprintf("http://%s:%d/health", host, svc.Port)
}
