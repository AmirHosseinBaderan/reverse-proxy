package config

import (
	"os"
	"reverse-proxy/internal/models/global"

	"gopkg.in/yaml.v3"
)

func LoadSettings(path string) (*global.Settings, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg global.Settings
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	if cfg.Server.Listen == "" {
		cfg.Server.Listen = ":80"
	}

	return &cfg, nil
}
