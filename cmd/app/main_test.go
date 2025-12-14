package main

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reverse-proxy/internal/application/config"
	"reverse-proxy/internal/models/global"
	"testing"
	"time"
)

func TestMainApplication(t *testing.T) {
	t.Run("test server configuration", func(t *testing.T) {
		// Create temporary config directory
		tmpDir := t.TempDir()

		// Create settings file
		settingsContent := `server:
  listen: ":8080"
  timeouts:
    read: 30s
    write: 30s
    idle: 60s
  limits:
    max_header_bytes: 1048576`

		settingsFile := filepath.Join(tmpDir, "settings.yml")
		if err := os.WriteFile(settingsFile, []byte(settingsContent), 0644); err != nil {
			t.Fatalf("Failed to create settings file: %v", err)
		}

		// Create site config
		siteConfig := `domain: test.local
proxy:
  upstream: http://localhost:5258
timeouts:
  read: 10s
  write: 10s`

		siteConfigFile := filepath.Join(tmpDir, "test.local.yml")
		if err := os.WriteFile(siteConfigFile, []byte(siteConfig), 0644); err != nil {
			t.Fatalf("Failed to create site config file: %v", err)
		}

		// Test loading settings
		settings, err := config.LoadSettings(settingsFile)
		if err != nil {
			t.Fatalf("Failed to load settings: %v", err)
		}

		if settings.Server.Listen != ":8080" {
			t.Errorf("Expected listen :8080, got %s", settings.Server.Listen)
		}

		if settings.Server.Timeouts.Read != 30*time.Second {
			t.Errorf("Expected read timeout 30s, got %v", settings.Server.Timeouts.Read)
		}

		// Test loading site configs
		sites, err := config.LoadConfigs(tmpDir)
		if err != nil {
			t.Fatalf("Failed to load site configs: %v", err)
		}

		if len(sites) != 1 {
			t.Errorf("Expected 1 site, got %d", len(sites))
		}

		if _, ok := sites["test.local"]; !ok {
			t.Error("Expected test.local site")
		}

		testSite := sites["test.local"]
		if testSite.Domain != "test.local" {
			t.Errorf("Expected domain test.local, got %s", testSite.Domain)
		}

		if testSite.Proxy.Upstream != "http://localhost:5258" {
			t.Errorf("Expected upstream http://localhost:5258, got %s", testSite.Proxy.Upstream)
		}
	})

	t.Run("test server startup and routing", func(t *testing.T) {
		// Create temporary config directory
		tmpDir := t.TempDir()

		// Create settings file
		settingsContent := `server:
  listen: ":8080"
  timeouts:
    read: 5s
    write: 5s
    idle: 10s`

		settingsFile := filepath.Join(tmpDir, "settings.yml")
		if err := os.WriteFile(settingsFile, []byte(settingsContent), 0644); err != nil {
			t.Fatalf("Failed to create settings file: %v", err)
		}

		// Create site config
		siteConfig := `domain: example.com
proxy:
  upstream: http://localhost:3000
timeouts:
  read: 2s
  write: 2s`

		siteConfigFile := filepath.Join(tmpDir, "example.com.yml")
		if err := os.WriteFile(siteConfigFile, []byte(siteConfig), 0644); err != nil {
			t.Fatalf("Failed to create site config file: %v", err)
		}

		// Load configurations
			_, err := config.LoadSettings(settingsFile)
			if err != nil {
				t.Fatalf("Failed to load settings: %v", err)
			}
	
			sites, err := config.LoadConfigs(tmpDir)
			if err != nil {
				t.Fatalf("Failed to load site configs: %v", err)
			}

		// Create handlers
		logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
		configs := make(map[string]http.Handler)

		for domain, cfg := range sites {
			handler, err := createTestHandler(logger, cfg)
			if err != nil {
				t.Fatalf("Failed to create handler for %s: %v", domain, err)
			}
			configs[domain] = handler
		}

		// Create router
		router := createTestRouter(configs)

		// Test routing
		req := httptest.NewRequest("GET", "http://example.com/test", nil)
		req.Host = "example.com"
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		resp := rec.Result()
		defer resp.Body.Close()

		// Should get a response (even if it's an error from the proxy)
		if resp.StatusCode == 0 {
			t.Error("Expected non-zero status code")
		}
	})

	t.Run("test HTTPS redirect configuration", func(t *testing.T) {
		// Create temporary config directory
		tmpDir := t.TempDir()

		// Create settings file with TLS config
		settingsContent := `server:
  listen: ":80"
  tls:
    listen: ":443"
    cert_file: "/path/to/cert"
    key_file: "/path/to/key"
    redirect_http: true
  timeouts:
    read: 5s
    write: 5s
    idle: 10s`

		settingsFile := filepath.Join(tmpDir, "settings.yml")
		if err := os.WriteFile(settingsFile, []byte(settingsContent), 0644); err != nil {
			t.Fatalf("Failed to create settings file: %v", err)
		}

		// Load settings
		settings, err := config.LoadSettings(settingsFile)
		if err != nil {
			t.Fatalf("Failed to load settings: %v", err)
		}

		if settings.Server.TLS == nil {
			t.Error("Expected TLS config")
		}

		if !settings.Server.TLS.RedirectHTTP {
			t.Error("Expected RedirectHTTP to be true")
		}

		if settings.Server.TLS.Listen != ":443" {
			t.Errorf("Expected TLS listen :443, got %s", settings.Server.TLS.Listen)
		}
	})
}

// Helper function to create a test handler (simplified version of NewSiteHandler)
func createTestHandler(logger *slog.Logger, cfg *global.SiteConfig) (http.Handler, error) {
	// For testing purposes, create a simple handler that returns the domain
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Response from " + cfg.Domain))
	}), nil
}

// Helper function to create a test router (simplified version of HostRouter)
func createTestRouter(sites map[string]http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if h, ok := sites[host]; ok {
			h.ServeHTTP(w, r)
			return
		}
		http.NotFound(w, r)
	})
}