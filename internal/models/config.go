package models

import "time"

type SiteConfig struct {
	Domain string `yaml:"domain"`
	Listen string `yaml:"listen"`

	Proxy struct {
		Upstream string            `yaml:"upstream"`
		Headers  map[string]string `yaml:"headers"`
	} `yaml:"proxy"`

	Timeouts struct {
		Read  time.Duration `yaml:"read"`
		Write time.Duration `yaml:"write"`
	} `yaml:"timeouts"`
}
