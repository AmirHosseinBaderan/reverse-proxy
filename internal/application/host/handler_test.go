package host

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHostRouter(t *testing.T) {
	t.Run("route to correct handler", func(t *testing.T) {
		// Create test handlers
		exampleHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Example site"))
		})

		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Test site"))
		})

		sites := map[string]http.Handler{
			"example.com": exampleHandler,
			"test.local":  testHandler,
		}

		router := HostRouter(sites)

		// Test request to example.com
		req := httptest.NewRequest("GET", "http://example.com/path", nil)
		req.Host = "example.com"
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		resp := rec.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200 for example.com, got %d", resp.StatusCode)
		}

		body := rec.Body.String()
		if body != "Example site" {
			t.Errorf("Expected 'Example site', got '%s'", body)
		}

		// Test request to test.local
		req2 := httptest.NewRequest("GET", "http://test.local/path", nil)
		req2.Host = "test.local"
		rec2 := httptest.NewRecorder()

		router.ServeHTTP(rec2, req2)

		resp2 := rec2.Result()
		defer resp2.Body.Close()

		if resp2.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200 for test.local, got %d", resp2.StatusCode)
		}

		body2 := rec2.Body.String()
		if body2 != "Test site" {
			t.Errorf("Expected 'Test site', got '%s'", body2)
		}
	})

	t.Run("unknown host returns 404", func(t *testing.T) {
		sites := map[string]http.Handler{
			"example.com": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}),
		}

		router := HostRouter(sites)

		// Test request to unknown host
		req := httptest.NewRequest("GET", "http://unknown.com/path", nil)
		req.Host = "unknown.com"
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		resp := rec.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status 404 for unknown host, got %d", resp.StatusCode)
		}
	})

	t.Run("host with port number", func(t *testing.T) {
		exampleHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Example site"))
		})

		sites := map[string]http.Handler{
			"example.com": exampleHandler,
		}

		router := HostRouter(sites)

		// Test request with port number
		req := httptest.NewRequest("GET", "http://example.com:8080/path", nil)
		req.Host = "example.com:8080"
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		resp := rec.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200 for example.com:8080, got %d", resp.StatusCode)
		}

		body := rec.Body.String()
		if body != "Example site" {
			t.Errorf("Expected 'Example site', got '%s'", body)
		}
	})

	t.Run("empty sites map", func(t *testing.T) {
		sites := map[string]http.Handler{}
		router := HostRouter(sites)

		req := httptest.NewRequest("GET", "http://example.com/path", nil)
		req.Host = "example.com"
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		resp := rec.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status 404 for empty sites map, got %d", resp.StatusCode)
		}
	})
}