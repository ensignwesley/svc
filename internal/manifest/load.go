package manifest

import (
	"fmt"
	"os"
	"sort"

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
		if svc.Port == 0 && svc.HealthURL == "" {
			result.Errors = append(result.Errors,
				fmt.Sprintf("service %q: one of 'port' or 'health_url' is required", id))
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
