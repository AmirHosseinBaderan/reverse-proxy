package global

import "time"

type Settings struct {
	Server Server `yaml:"server"`
}

type Server struct {
	Listen   string   `yaml:"listen"`
	Timeouts Timeouts `yaml:"timeouts"`
	Limits   Limits   `yaml:"limits"`
}

type Timeouts struct {
	Read  time.Duration `yaml:"read"`
	Write time.Duration `yaml:"write"`
	Idle  time.Duration `yaml:"idle"`
}

type Limits struct {
	MaxHeaderBytes int `yaml:"max_header_bytes"`
}
