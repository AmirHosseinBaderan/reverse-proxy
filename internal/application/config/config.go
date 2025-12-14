package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"reverse-proxy/internal/models"
	"strings"
)

func LoadConfigs(dir string) (map[string]*models.SiteConfig, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	sites := make(map[string]*models.SiteConfig)

	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".yml") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, f.Name()))
		if err != nil {
			return nil, err
		}

		var cfg models.SiteConfig
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, err
		}

		if cfg.Domain == "" {
			return nil, fmt.Errorf("domain missing in %s", f.Name())
		}

		sites[cfg.Domain] = &cfg
	}

	return sites, nil
}
