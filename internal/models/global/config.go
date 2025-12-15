package global

type SiteConfig struct {
	Domain   string   `yaml:"domain"`
	Listen   string   `yaml:"listen"`
	Proxy    Proxy    `yaml:"proxy"`
	Timeouts Timeouts `yaml:"timeouts"`
}

type Proxy struct {
	Upstream        string                 `yaml:"upstream"`
	Upstreams       []string               `yaml:"upstreams"`
	LoadBalance     *LoadBalance           `yaml:"load_balance"`
	Headers         map[string]string      `yaml:"headers"`
}

type LoadBalance struct {
	Algorithm      string                 `yaml:"algorithm"`
	HealthCheck    *HealthCheck           `yaml:"health_check"`
}

type HealthCheck struct {
	Path           string                 `yaml:"path"`
	Interval       string                 `yaml:"interval"`
	Timeout        string                 `yaml:"timeout"`
}
