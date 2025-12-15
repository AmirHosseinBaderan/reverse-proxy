// Package global contains shared data models and configuration structures
// used throughout the reverse proxy application.
package global

// SiteConfig represents the configuration for a single proxy site.
type SiteConfig struct {
	Domain   string   `yaml:"domain"`
	Listen   string   `yaml:"listen"`
	Proxy    Proxy    `yaml:"proxy"`
	Timeouts Timeouts `yaml:"timeouts"`
}

type Proxy struct {
	PathBase
	Paths []ProxyPath `yaml:"paths"`
}

type ProxyPath struct {
	PathBase
	Path string `yaml:"path"`
}

type LoadBalance struct {
	Algorithm   string       `yaml:"algorithm"`
	HealthCheck *HealthCheck `yaml:"health_check"`
}

type HealthCheck struct {
	Path     string `yaml:"path"`
	Interval string `yaml:"interval"`
	Timeout  string `yaml:"timeout"`
}

type PathBase struct {
	Upstream    string            `yaml:"upstream"`
	Upstreams   []string          `yaml:"upstreams"`
	Headers     map[string]string `yaml:"headers"`
	LoadBalance *LoadBalance      `yaml:"load_balance"`
}
