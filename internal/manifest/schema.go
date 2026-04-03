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
	History     History  `yaml:"history"`
}

// History holds optional history retention settings.
type History struct {
	// Retention is the maximum age of check rows to keep.
	// Format: "90d", "30d", "7d", etc. Empty means no auto-prune.
	// Incidents are never auto-pruned regardless of this setting.
	Retention string `yaml:"retention"`
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
