package site

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reverse-proxy/internal/models/global"
	"testing"
	"time"
)

func TestNewSiteHandler(t *testing.T) {
	t.Run("valid configuration", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
		
		cfg := &global.SiteConfig{
			Domain: "example.com",
			Proxy: global.Proxy{
				Upstream: "http://localhost:3000",
				Headers: map[string]string{
					"X-Forwarded-For": "$remote_addr",
					"X-Custom-Header": "test-value",
				},
			},
			Timeouts: global.Timeouts{
				Read:  10 * time.Second,
				Write: 10 * time.Second,
			},
		}

		handler, err := NewSiteHandler(logger, cfg)
		if err != nil {
			t.Fatalf("NewSiteHandler failed: %v", err)
		}

		if handler == nil {
			t.Error("Expected non-nil handler")
		}

		if handler.Site != cfg {
			t.Error("Expected handler Site to match config")
		}

		if handler.Logger != logger {
			t.Error("Expected handler Logger to match input logger")
		}

		if handler.Handler == nil {
			t.Error("Expected handler Handler to be non-nil")
		}
	})

	t.Run("invalid upstream URL", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
		
		cfg := &global.SiteConfig{
			Domain: "example.com",
			Proxy: global.Proxy{
				Upstream: "://invalid-url-without-scheme",
			},
		}

		_, err := NewSiteHandler(logger, cfg)
		if err == nil {
			t.Error("Expected error for invalid upstream URL, got nil")
		}
	})

	t.Run("empty domain", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
		
		cfg := &global.SiteConfig{
			Domain: "",
			Proxy: global.Proxy{
				Upstream: "http://localhost:3000",
			},
		}

		handler, err := NewSiteHandler(logger, cfg)
		if err != nil {
			t.Fatalf("NewSiteHandler failed: %v", err)
		}

		// Should still work with empty domain, just not ideal
		if handler == nil {
			t.Error("Expected non-nil handler even with empty domain")
		}
	})
}

func TestSiteHandlerServeHTTP(t *testing.T) {
	t.Run("successful proxy request", func(t *testing.T) {
		// Create a test upstream server
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Hello from upstream"))
		}))
		defer upstream.Close()

		logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
		
		cfg := &global.SiteConfig{
			Domain: "example.com",
			Proxy: global.Proxy{
				Upstream: upstream.URL,
				Headers: map[string]string{
					"X-Test-Header": "test-value",
				},
			},
			Timeouts: global.Timeouts{
				Read:  5 * time.Second,
				Write: 5 * time.Second,
			},
		}

		handler, err := NewSiteHandler(logger, cfg)
		if err != nil {
			t.Fatalf("NewSiteHandler failed: %v", err)
		}

		// Create a request to the handler
		req := httptest.NewRequest("GET", "http://example.com/test", nil)
		rec := httptest.NewRecorder()

		handler.Handler.ServeHTTP(rec, req)

		resp := rec.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		// Note: Custom headers are added to the upstream request, not the original request
		// This test verifies that the handler works correctly by checking the response
		body := rec.Body.String()
		if body != "Hello from upstream" {
			t.Errorf("Expected 'Hello from upstream', got '%s'", body)
		}
	})

	t.Run("upstream server error", func(t *testing.T) {
		// Create a test upstream server that returns an error
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Internal Server Error"))
		}))
		defer upstream.Close()

		logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
		
		cfg := &global.SiteConfig{
			Domain: "example.com",
			Proxy: global.Proxy{
				Upstream: upstream.URL,
			},
			Timeouts: global.Timeouts{
				Read:  5 * time.Second,
				Write: 5 * time.Second,
			},
		}

		handler, err := NewSiteHandler(logger, cfg)
		if err != nil {
			t.Fatalf("NewSiteHandler failed: %v", err)
		}

		// Create a request to the handler
		req := httptest.NewRequest("GET", "http://example.com/test", nil)
		rec := httptest.NewRecorder()

		handler.Handler.ServeHTTP(rec, req)

		resp := rec.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", resp.StatusCode)
		}
	})

	t.Run("timeout handling", func(t *testing.T) {
		// Create a slow upstream server
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Slow response"))
		}))
		defer upstream.Close()

		logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
		
		cfg := &global.SiteConfig{
			Domain: "example.com",
			Proxy: global.Proxy{
				Upstream: upstream.URL,
			},
			Timeouts: global.Timeouts{
				Read:  1 * time.Millisecond,  // Very short timeout
				Write: 1 * time.Millisecond,
			},
		}

		handler, err := NewSiteHandler(logger, cfg)
		if err != nil {
			t.Fatalf("NewSiteHandler failed: %v", err)
		}

		// Create a request to the handler
		req := httptest.NewRequest("GET", "http://example.com/test", nil)
		rec := httptest.NewRecorder()

		handler.Handler.ServeHTTP(rec, req)

		resp := rec.Result()
		defer resp.Body.Close()

		// Should get timeout response
		if resp.StatusCode != http.StatusServiceUnavailable {
			t.Errorf("Expected timeout status 503, got %d", resp.StatusCode)
		}
	})
}

func TestResponseWriter(t *testing.T) {
	t.Run("write header and data", func(t *testing.T) {
		w := httptest.NewRecorder()
		rw := &responseWriter{ResponseWriter: w}

		// Write some data without explicitly setting header
		_, err := rw.Write([]byte("test data"))
		if err != nil {
			t.Fatalf("Write failed: %v", err)
		}

		if rw.status != 200 {
			t.Errorf("Expected status 200, got %d", rw.status)
		}

		if rw.size != 9 {
			t.Errorf("Expected size 9, got %d", rw.size)
		}

		resp := w.Result()
		if resp.StatusCode != 200 {
			t.Errorf("Expected response status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("explicit header then data", func(t *testing.T) {
		w := httptest.NewRecorder()
		rw := &responseWriter{ResponseWriter: w}

		rw.WriteHeader(http.StatusNotFound)
		_, err := rw.Write([]byte("not found"))
		if err != nil {
			t.Fatalf("Write failed: %v", err)
		}

		if rw.status != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", rw.status)
		}

		resp := w.Result()
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected response status 404, got %d", resp.StatusCode)
		}
	})
}