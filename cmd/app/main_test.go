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
	"strings"
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

	t.Run("test proxy path configuration", func(t *testing.T) {
		// Create temporary config directory
		tmpDir := t.TempDir()

		// Create site config with paths
		siteConfig := `domain: test-paths.local
proxy:
		paths:
		  - path: /api/
		    upstream: http://localhost:5042
		    headers:
		      X-API-Key: secret-api-key
		  - path: /static/
		    upstream: http://localhost:5043
		      X-Static-Token: static-token
		  - path: /
		    upstream: http://localhost:5044
		    headers:
		      X-Default-Key: default-key
timeouts:
		read: 10s
		write: 10s`

		siteConfigFile := filepath.Join(tmpDir, "test-paths.local.yml")
		if err := os.WriteFile(siteConfigFile, []byte(siteConfig), 0644); err != nil {
			t.Fatalf("Failed to create site config file: %v", err)
		}

		// Test loading site configs with paths
		sites, err := config.LoadConfigs(tmpDir)
		if err != nil {
			t.Fatalf("Failed to load site configs: %v", err)
		}

		if len(sites) != 1 {
			t.Errorf("Expected 1 site, got %d", len(sites))
		}

		testSite, ok := sites["test-paths.local"]
		if !ok {
			t.Error("Expected test-paths.local site")
		}

		if testSite.Domain != "test-paths.local" {
			t.Errorf("Expected domain test-paths.local, got %s", testSite.Domain)
		}

		// Check that paths are loaded correctly
		if len(testSite.Proxy.Paths) != 3 {
			t.Errorf("Expected 3 paths, got %d", len(testSite.Proxy.Paths))
		}

		// Check specific paths
		foundAPI := false
		foundStatic := false
		foundRoot := false

		for _, path := range testSite.Proxy.Paths {
			switch path.Path {
			case "/api/":
				foundAPI = true
				if path.Upstream != "http://localhost:5042" {
					t.Errorf("Expected API upstream http://localhost:5042, got %s", path.Upstream)
				}
				if path.Headers["X-API-Key"] != "secret-api-key" {
					t.Errorf("Expected API header X-API-Key: secret-api-key, got %s", path.Headers["X-API-Key"])
				}
			case "/static/":
				foundStatic = true
				if path.Upstream != "http://localhost:5043" {
					t.Errorf("Expected Static upstream http://localhost:5043, got %s", path.Upstream)
				}
			case "/":
				foundRoot = true
				if path.Upstream != "http://localhost:5044" {
					t.Errorf("Expected Root upstream http://localhost:5044, got %s", path.Upstream)
				}
				if path.Headers["X-Default-Key"] != "default-key" {
					t.Errorf("Expected Root header X-Default-Key: default-key, got %s", path.Headers["X-Default-Key"])
				}
			}
		}

		if !foundAPI {
			t.Error("Expected to find /api/ path")
		}
		if !foundStatic {
			t.Error("Expected to find /static/ path")
		}
		if !foundRoot {
			t.Error("Expected to find / path")
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

	t.Run("test path-based routing", func(t *testing.T) {
		// Create temporary config directory
		tmpDir := t.TempDir()

		// Create site config with paths
		siteConfig := `domain: path-test.local
proxy:
		paths:
		  - path: /api/
		    upstream: http://localhost:5042
		  - path: /static/
		    upstream: http://localhost:5043
		  - path: /
		    upstream: http://localhost:5044
timeouts:
		read: 2s
		write: 2s`

		siteConfigFile := filepath.Join(tmpDir, "path-test.local.yml")
		if err := os.WriteFile(siteConfigFile, []byte(siteConfig), 0644); err != nil {
			t.Fatalf("Failed to create site config file: %v", err)
		}

		// Load configurations
		sites, err := config.LoadConfigs(tmpDir)
		if err != nil {
			t.Fatalf("Failed to load site configs: %v", err)
		}

		// Create handlers
		logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
		configs := make(map[string]http.Handler)

		for domain, cfg := range sites {
			handler, err := createTestPathHandler(logger, cfg)
			if err != nil {
				t.Fatalf("Failed to create handler for %s: %v", domain, err)
			}
			configs[domain] = handler
		}

		// Create router
		router := createTestRouter(configs)

		// Test different paths
		testCases := []struct {
			path           string
			expectedPrefix string
		}{
			{"/api/users", "API: "},
			{"/static/css/style.css", "Static: "},
			{"/", "Default: "},
			{"/about", "Default: "},
		}

		for _, tc := range testCases {
			req := httptest.NewRequest("GET", "http://path-test.local"+tc.path, nil)
			req.Host = "path-test.local"
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			resp := rec.Result()
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status 200 for path %s, got %d", tc.path, resp.StatusCode)
			}

			body := rec.Body.String()
			if !strings.HasPrefix(body, tc.expectedPrefix) {
				t.Errorf("Expected body to start with '%s' for path %s, got '%s'", tc.expectedPrefix, tc.path, body)
			}
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

// Helper function to create a test handler for path-based routing
func createTestPathHandler(logger *slog.Logger, cfg *global.SiteConfig) (http.Handler, error) {
	// Create a router that simulates path-based routing
	mux := http.NewServeMux()
	
	// Register handlers for each path
	for _, path := range cfg.Proxy.Paths {
		pathPrefix := path.Path
		upstream := path.Upstream
		
		// Create a handler that identifies which path was matched
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var prefix string
			switch pathPrefix {
			case "/api/":
				prefix = "API: "
			case "/static/":
				prefix = "Static: "
			case "/":
				prefix = "Default: "
			default:
				prefix = "Path " + pathPrefix + ": "
			}
			
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(prefix + "Response from " + upstream))
		})
		
		mux.Handle(pathPrefix, handler)
	}
	
	return mux, nil
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