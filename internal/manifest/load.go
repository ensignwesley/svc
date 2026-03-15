package manifest

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Load reads and validates a manifest from the given file path.
func Load(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", path, err)
	}

	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("invalid YAML in %s: %w", path, err)
	}

	if m.Meta.Version == 0 {
		m.Meta.Version = 1
	}
	if m.Meta.Host == "" {
		m.Meta.Host = "localhost"
	}
	if m.Services == nil {
		m.Services = make(map[string]Service)
	}

	if err := validate(&m); err != nil {
		return nil, err
	}

	return &m, nil
}

func validate(m *Manifest) error {
	for id, svc := range m.Services {
		if svc.Port == 0 && svc.HealthURL == "" {
			return fmt.Errorf("service %q: must set either 'port' or 'health_url'", id)
		}
	}
	return nil
}
