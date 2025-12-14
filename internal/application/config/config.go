package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reverse-proxy/internal/models/global"
	"strings"

	"gopkg.in/yaml.v3"
)

func LoadConfigs(dir string) (map[string]*global.SiteConfig, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	sites := make(map[string]*global.SiteConfig)

	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".yml") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, f.Name()))
		if err != nil {
			return nil, err
		}

		var cfg global.SiteConfig
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
