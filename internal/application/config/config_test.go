package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadSettings(t *testing.T) {
	t.Run("valid settings", func(t *testing.T) {
		// Create temporary settings file
		settingsContent := `server:
  listen: ":8080"
  timeouts:
    read: 30s
    write: 30s
    idle: 60s
  limits:
    max_header_bytes: 1048576
  tls:
    listen: ":443"
    cert_file: "/path/to/cert"
    key_file: "/path/to/key"
    redirect_http: true`

		tmpFile := createTempFile(t, settingsContent)
		defer os.Remove(tmpFile)

		settings, err := LoadSettings(tmpFile)
		if err != nil {
			t.Fatalf("LoadSettings failed: %v", err)
		}

		if settings.Server.Listen != ":8080" {
			t.Errorf("Expected listen :8080, got %s", settings.Server.Listen)
		}

		if settings.Server.TLS == nil {
			t.Error("Expected TLS config, got nil")
		} else {
			if settings.Server.TLS.Listen != ":443" {
				t.Errorf("Expected TLS listen :443, got %s", settings.Server.TLS.Listen)
			}
			if !settings.Server.TLS.RedirectHTTP {
				t.Error("Expected RedirectHTTP true, got false")
			}
		}
	})

	t.Run("default listen port", func(t *testing.T) {
		// Create settings file without listen port
		settingsContent := `server:
  timeouts:
    read: 30s
    write: 30s
    idle: 60s`

		tmpFile := createTempFile(t, settingsContent)
		defer os.Remove(tmpFile)

		settings, err := LoadSettings(tmpFile)
		if err != nil {
			t.Fatalf("LoadSettings failed: %v", err)
		}

		if settings.Server.Listen != ":80" {
			t.Errorf("Expected default listen :80, got %s", settings.Server.Listen)
		}
	})

	t.Run("invalid file", func(t *testing.T) {
		_, err := LoadSettings("/nonexistent/file.yml")
		if err == nil {
			t.Error("Expected error for nonexistent file, got nil")
		}
	})

	t.Run("invalid yaml", func(t *testing.T) {
		invalidContent := `server:
  listen: ":8080"
  invalid: [unclosed bracket`

		tmpFile := createTempFile(t, invalidContent)
		defer os.Remove(tmpFile)

		_, err := LoadSettings(tmpFile)
		if err == nil {
			t.Error("Expected error for invalid YAML, got nil")
		}
	})
}

func TestLoadConfigs(t *testing.T) {
	t.Run("valid configs", func(t *testing.T) {
		// Create temporary directory with config files
		tmpDir := t.TempDir()

		// Create first site config
		config1 := `domain: example.com
proxy:
  upstream: http://localhost:3000
  headers:
    X-Forwarded-For: $remote_addr
timeouts:
  read: 10s
  write: 10s`

		config1File := filepath.Join(tmpDir, "example.com.yml")
		if err := os.WriteFile(config1File, []byte(config1), 0644); err != nil {
			t.Fatalf("Failed to create config file: %v", err)
		}

		// Create second site config
		config2 := `domain: test.local
proxy:
  upstream: http://localhost:4000
timeouts:
  read: 5s
  write: 5s`

		config2File := filepath.Join(tmpDir, "test.local.yml")
		if err := os.WriteFile(config2File, []byte(config2), 0644); err != nil {
			t.Fatalf("Failed to create config file: %v", err)
		}

		// Create settings.yml (should be ignored)
		settingsContent := `server:
  listen: ":8080"`
		settingsFile := filepath.Join(tmpDir, "settings.yml")
		if err := os.WriteFile(settingsFile, []byte(settingsContent), 0644); err != nil {
			t.Fatalf("Failed to create settings file: %v", err)
		}

		sites, err := LoadConfigs(tmpDir)
		if err != nil {
			t.Fatalf("LoadConfigs failed: %v", err)
		}

		if len(sites) != 2 {
			t.Errorf("Expected 2 sites, got %d", len(sites))
		}

		if _, ok := sites["example.com"]; !ok {
			t.Error("Expected example.com site")
		}

		if _, ok := sites["test.local"]; !ok {
			t.Error("Expected test.local site")
		}

		exampleSite := sites["example.com"]
		if exampleSite.Domain != "example.com" {
			t.Errorf("Expected domain example.com, got %s", exampleSite.Domain)
		}

		if exampleSite.Proxy.Upstream != "http://localhost:3000" {
			t.Errorf("Expected upstream http://localhost:3000, got %s", exampleSite.Proxy.Upstream)
		}

		if len(exampleSite.Proxy.Headers) != 1 {
			t.Errorf("Expected 1 header, got %d", len(exampleSite.Proxy.Headers))
		}

		if exampleSite.Timeouts.Read != 10*time.Second {
			t.Errorf("Expected read timeout 10s, got %v", exampleSite.Timeouts.Read)
		}
	})

	t.Run("missing domain", func(t *testing.T) {
		tmpDir := t.TempDir()

		invalidConfig := `proxy:
  upstream: http://localhost:3000`

		configFile := filepath.Join(tmpDir, "invalid.yml")
		if err := os.WriteFile(configFile, []byte(invalidConfig), 0644); err != nil {
			t.Fatalf("Failed to create config file: %v", err)
		}

		_, err := LoadConfigs(tmpDir)
		if err == nil {
			t.Error("Expected error for missing domain, got nil")
		}
	})

	t.Run("invalid yaml in config", func(t *testing.T) {
		tmpDir := t.TempDir()

		invalidConfig := `domain: example.com
proxy:
  upstream: [invalid yaml`

		configFile := filepath.Join(tmpDir, "invalid.yml")
		if err := os.WriteFile(configFile, []byte(invalidConfig), 0644); err != nil {
			t.Fatalf("Failed to create config file: %v", err)
		}

		_, err := LoadConfigs(tmpDir)
		if err == nil {
			t.Error("Expected error for invalid YAML in config, got nil")
		}
	})

	t.Run("no config files", func(t *testing.T) {
		tmpDir := t.TempDir()

		sites, err := LoadConfigs(tmpDir)
		if err != nil {
			t.Fatalf("LoadConfigs failed: %v", err)
		}

		if len(sites) != 0 {
			t.Errorf("Expected 0 sites, got %d", len(sites))
		}
	})
}

// Helper function to create temporary files
func createTempFile(t *testing.T, content string) string {
	tmpFile, err := os.CreateTemp("", "settings_test_*.yml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
		}
	
	if _, err := tmpFile.Write([]byte(content)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
		}
	
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
		}
	
	return tmpFile.Name()
}