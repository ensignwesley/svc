package manifest

// Manifest is the top-level structure parsed from services.yaml.
type Manifest struct {
	Meta     Meta               `yaml:"manifest"`
	Services map[string]Service `yaml:"services"`
}

// Meta holds global defaults for the manifest.
type Meta struct {
	Version int    `yaml:"version"`
	Host    string `yaml:"host"`
}

// Service describes a single service entry.
type Service struct {
	Description  string   `yaml:"description"`
	Port         int      `yaml:"port"`
	HealthURL    string   `yaml:"health_url"`
	SystemdUnit  string   `yaml:"systemd_unit"`
	Repo         string   `yaml:"repo"`
	Version      string   `yaml:"version"`
	MaxMajor     int      `yaml:"max_major"`
	Docs         string   `yaml:"docs"`
	Tags         []string `yaml:"tags"`
	Added        string   `yaml:"added"`
}

// ResolvedHealthURL returns the health URL for this service.
// If health_url is set, it's used directly.
// Otherwise derived from host:port/health.
func (s *Service) ResolvedHealthURL(host string) string {
	if s.HealthURL != "" {
		return s.HealthURL
	}
	if host == "" {
		host = "localhost"
	}
	return "http://" + host + ":" + itoa(s.Port) + "/health"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := make([]byte, 0, 10)
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	return string(buf)
}
