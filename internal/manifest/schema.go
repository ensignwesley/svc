package manifest

// Manifest is the top-level structure of services.yaml.
type Manifest struct {
	Meta     Meta               `yaml:"manifest"`
	Services map[string]Service `yaml:"services"`
}

// Meta holds global manifest settings.
type Meta struct {
	Version     int      `yaml:"version"`
	Host        string   `yaml:"host"`
	IgnoreUnits []string `yaml:"ignore_units"`
}

// Service describes a single self-hosted service.
type Service struct {
	Description  string   `yaml:"description"`
	Host         string   `yaml:"host"`       // optional; SSH host for remote systemd checks
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
