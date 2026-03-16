package manifest

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

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

	if err := validate(&m); err != nil {
		return nil, err
	}

	// Apply defaults.
	if m.Meta.Host == "" {
		m.Meta.Host = "localhost"
	}

	return &m, nil
}

// validate checks manifest semantics after YAML parsing.
func validate(m *Manifest) error {
	if m.Meta.Version != 1 {
		if m.Meta.Version == 0 {
			return fmt.Errorf("manifest.version is required (set to 1)")
		}
		return fmt.Errorf("unsupported manifest version %d (expected 1)", m.Meta.Version)
	}

	for id, svc := range m.Services {
		if svc.Port == 0 && svc.HealthURL == "" {
			return fmt.Errorf("service %q: one of 'port' or 'health_url' is required", id)
		}
		if svc.Repo != "" && svc.Version == "" {
			// Warn-only: version check will be skipped.
			// We don't error here — caller can warn separately.
		}
	}

	return nil
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
