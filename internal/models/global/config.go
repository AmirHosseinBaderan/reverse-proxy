package global

type SiteConfig struct {
	Domain   string   `yaml:"domain"`
	Listen   string   `yaml:"listen"`
	Proxy    Proxy    `yaml:"proxy"`
	Timeouts Timeouts `yaml:"timeouts"`
}

type Proxy struct {
	Upstream string            `yaml:"upstream"`
	Headers  map[string]string `yaml:"headers"`
}
