// Package site provides HTTP handler implementations for reverse proxy sites.
// It supports single upstream proxies, load-balanced proxies, and path-based routing.
package site

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"reverse-proxy/internal/models/global"
	"sync"
	"sync/atomic"
	"time"
)

func Max(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
}

type responseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(data []byte) (int, error) {
	if rw.status == 0 {
		rw.status = 200
	}
	size, err := rw.ResponseWriter.Write(data)
	rw.size += size
	return size, err
}

func loggingHandler(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		logger.Info("Request", "method", r.Method, "url", r.URL.String(), "remote", r.RemoteAddr, "host", r.Host, "user_agent", r.UserAgent())
		wrapped := &responseWriter{ResponseWriter: w}
		next.ServeHTTP(wrapped, r)
		duration := time.Since(start)
		logger.Info("Response", "status", wrapped.status, "size", wrapped.size, "duration", duration)
	})
}

type Handler struct {
	Site    *global.SiteConfig
	Handler http.Handler
	Logger  *slog.Logger
	lb      *LoadBalancer
}

type LoadBalancer struct {
	upstreams    []*url.URL
	algorithm    string
	currentIndex uint64
	mu           sync.Mutex
}

func NewLoadBalancer(upstreams []string, algorithm string) (*LoadBalancer, error) {
	lb := &LoadBalancer{
		algorithm: algorithm,
	}

	for _, upstream := range upstreams {
		u, err := url.Parse(upstream)
		if err != nil {
			return nil, fmt.Errorf("invalid upstream URL: %s", upstream)
		}
		lb.upstreams = append(lb.upstreams, u)
	}

	if len(lb.upstreams) == 0 {
		return nil, fmt.Errorf("no valid upstreams provided")
	}

	return lb, nil
}

func (lb *LoadBalancer) Next() *url.URL {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	switch lb.algorithm {
	case "round-robin", "":
		// Round-robin algorithm
		index := atomic.AddUint64(&lb.currentIndex, 1) - 1
		return lb.upstreams[index%uint64(len(lb.upstreams))]
	case "random":
		// Random algorithm (simplified)
		index := time.Now().UnixNano() % int64(len(lb.upstreams))
		return lb.upstreams[index]
	default:
		// Default to round-robin
		index := atomic.AddUint64(&lb.currentIndex, 1) - 1
		return lb.upstreams[index%uint64(len(lb.upstreams))]
	}
}

func NewLoadBalancedProxy(lb *LoadBalancer, logger *slog.Logger, cfg *global.SiteConfig) http.Handler {
	transport := &http.Transport{
		MaxIdleConns:        1000,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Select ONE upstream server for this request
		target := lb.Next()

		// Clone the original request to avoid modifying it
		outReq := r.Clone(r.Context())
		outReq.URL.Scheme = target.Scheme
		outReq.URL.Host = target.Host
		// Keep the original request path
		outReq.URL.Path = r.URL.Path
		if target.RawPath != "" {
			outReq.URL.RawPath = target.RawPath
		}
		if target.RawQuery == "" || r.URL.RawQuery == "" {
			outReq.URL.RawQuery = target.RawQuery
		} else {
			outReq.URL.RawQuery = target.RawQuery + "&" + r.URL.RawQuery
		}

		// Add custom headers
		for k, v := range cfg.Proxy.Headers {
			outReq.Header.Set(k, v)
		}

		logger.Info("Load balanced request", "method", r.Method, "path", r.URL.Path, "target", target.String())

		// Make the request to the selected upstream
		resp, err := transport.RoundTrip(outReq)
		if err != nil {
			logger.Error("Upstream error", "domain", cfg.Domain, "url", r.URL.String(), "target", target.String(), "error", err)
			http.Error(w, "Upstream error", http.StatusBadGateway)
			return
		}
		defer func(Body io.ReadCloser) {
			_ = Body.Close()
		}(resp.Body)

		logger.Info("Upstream response", "status", resp.StatusCode, "size", resp.ContentLength, "target", target.String())

		// Copy response headers
		for key, values := range resp.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}

		// Copy status code and body
		w.WriteHeader(resp.StatusCode)
		if _, err := io.Copy(w, resp.Body); err != nil {
			logger.Error("Failed to copy response body", "error", err)
		}
	})
}

func NewLoadBalancedProxyWithHeaders(lb *LoadBalancer, logger *slog.Logger, cfg *global.SiteConfig, pathCfg *global.ProxyPath) http.Handler {
	transport := &http.Transport{
		MaxIdleConns:        1000,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Select ONE upstream server for this request
		target := lb.Next()

		// Clone the original request to avoid modifying it
		outReq := r.Clone(r.Context())
		outReq.URL.Scheme = target.Scheme
		outReq.URL.Host = target.Host
		// Keep the original request path
		outReq.URL.Path = r.URL.Path
		if target.RawPath != "" {
			outReq.URL.RawPath = target.RawPath
		}
		if target.RawQuery == "" || r.URL.RawQuery == "" {
			outReq.URL.RawQuery = target.RawQuery
		} else {
			outReq.URL.RawQuery = target.RawQuery + "&" + r.URL.RawQuery
		}

		// Add global headers first
		for k, v := range cfg.Proxy.Headers {
			outReq.Header.Set(k, v)
		}

		// Add path-specific headers
		for k, v := range pathCfg.Headers {
			outReq.Header.Set(k, v)
		}

		logger.Info("Load balanced request", "method", r.Method, "path", r.URL.Path, "target", target.String(), "path_config", pathCfg.Path)

		// Make the request to the selected upstream
		resp, err := transport.RoundTrip(outReq)
		if err != nil {
			logger.Error("Upstream error", "domain", cfg.Domain, "url", r.URL.String(), "target", target.String(), "path_config", pathCfg.Path, "error", err)
			http.Error(w, "Upstream error", http.StatusBadGateway)
			return
		}
		defer func(Body io.ReadCloser) {
			_ = Body.Close()
		}(resp.Body)

		logger.Info("Upstream response", "status", resp.StatusCode, "size", resp.ContentLength, "target", target.String(), "path_config", pathCfg.Path)

		// Copy response headers
		for key, values := range resp.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}

		// Copy status code and body
		w.WriteHeader(resp.StatusCode)
		if _, err := io.Copy(w, resp.Body); err != nil {
			logger.Error("Failed to copy response body", "error", err)
		}
	})
}

func NewSiteHandler(logger *slog.Logger, cfg *global.SiteConfig) (*Handler, error) {
	// Check if path-based routing is configured
	if len(cfg.Proxy.Paths) > 0 {
		mux := http.NewServeMux()
		var err error

		// Create a proxy for each path
		for i, pathCfg := range cfg.Proxy.Paths {
			var pathProxy http.Handler
			var pathLb *LoadBalancer

			// Check if this path has multiple upstreams (load balancing)
			if len(pathCfg.Upstreams) > 0 {
				// Path has its own upstreams - use path-specific load balancing
				algorithm := "round-robin"
				if pathCfg.LoadBalance != nil && pathCfg.LoadBalance.Algorithm != "" {
					algorithm = pathCfg.LoadBalance.Algorithm
				}

				pathLb, err = NewLoadBalancer(pathCfg.Upstreams, algorithm)
				if err != nil {
					return nil, fmt.Errorf("failed to create load balancer for path %s: %w", pathCfg.Path, err)
				}

				// Create a load-balanced proxy for this path with path-specific headers
				pathProxy = NewLoadBalancedProxyWithHeaders(pathLb, logger, cfg, &cfg.Proxy.Paths[i])
			} else if len(cfg.Proxy.Upstreams) > 0 {
				// Use global upstreams for this path
				algorithm := "round-robin"
				if cfg.Proxy.LoadBalance != nil && cfg.Proxy.LoadBalance.Algorithm != "" {
					algorithm = cfg.Proxy.LoadBalance.Algorithm
				}

				pathLb, err = NewLoadBalancer(cfg.Proxy.Upstreams, algorithm)
				if err != nil {
					return nil, fmt.Errorf("failed to create load balancer for path %s: %w", pathCfg.Path, err)
				}

				// Create a load-balanced proxy for this path with path-specific headers
				pathProxy = NewLoadBalancedProxyWithHeaders(pathLb, logger, cfg, &cfg.Proxy.Paths[i])
			} else {
				// Single upstream for this path
				target, err := url.Parse(pathCfg.Upstream)
				if err != nil {
					return nil, fmt.Errorf("invalid upstream for path %s: %w", pathCfg.Path, err)
				}

				transport := &http.Transport{
					MaxIdleConns:        1000,
					MaxIdleConnsPerHost: 100,
					IdleConnTimeout:     90 * time.Second,
				}

				proxy := httputil.NewSingleHostReverseProxy(target)
				proxy.Transport = transport

				// Handle headers for single upstream
				originalDirector := proxy.Director
				proxy.Director = func(req *http.Request) {
					originalDirector(req)
					// Apply global headers first
					for k, v := range cfg.Proxy.Headers {
						req.Header.Set(k, v)
					}
					// Apply path-specific headers
					for k, v := range pathCfg.Headers {
						req.Header.Set(k, v)
					}
				}

				pathProxy = proxy
			}

			// Apply per-site timeouts
			timeoutHandler := http.TimeoutHandler(
				pathProxy,
				Max(cfg.Timeouts.Read, cfg.Timeouts.Write),
				"Request timeout",
			)

			// Register the path with the router
			mux.Handle(pathCfg.Path, timeoutHandler)
			logger.Info("Registered path", "path", pathCfg.Path, "upstream", pathCfg.Upstream)
		}

		// Add request/response logging
		loggedHandler := loggingHandler(logger, mux)

		return &Handler{
			Site:    cfg,
			Handler: loggedHandler,
			Logger:  logger,
			lb:      nil, // No global load balancer when using paths
		}, nil
	}

	// Fallback to original behavior (single upstream or load balancing)
	var proxy http.Handler
	var lb *LoadBalancer
	var err error

	// Check if load balancing is configured
	if len(cfg.Proxy.Upstreams) > 0 {
		// Use load balancing
		algorithm := "round-robin"
		if cfg.Proxy.LoadBalance != nil && cfg.Proxy.LoadBalance.Algorithm != "" {
			algorithm = cfg.Proxy.LoadBalance.Algorithm
		}

		lb, err = NewLoadBalancer(cfg.Proxy.Upstreams, algorithm)
		if err != nil {
			return nil, fmt.Errorf("failed to create load balancer for %s: %w", cfg.Domain, err)
		}

		proxy = NewLoadBalancedProxy(lb, logger, cfg)
	} else if cfg.Proxy.Upstream != "" {
		// Use single upstream (backward compatibility)
		target, err := url.Parse(cfg.Proxy.Upstream)
		if err != nil {
			return nil, fmt.Errorf("invalid upstream for %s: %w", cfg.Domain, err)
		}

		transport := &http.Transport{
			MaxIdleConns:        1000,
			MaxIdleConnsPerHost: 100,
			IdleConnTimeout:     90 * time.Second,
		}

		reverseProxy := httputil.NewSingleHostReverseProxy(target)
		reverseProxy.Transport = transport

		// Handle headers for single upstream
		originalDirector := reverseProxy.Director
		reverseProxy.Director = func(req *http.Request) {
			originalDirector(req)
			for k, v := range cfg.Proxy.Headers {
				req.Header.Set(k, v)
			}
		}

		proxy = reverseProxy
	} else {
		return nil, fmt.Errorf("no upstream or upstreams configured for %s", cfg.Domain)
	}

	// Apply per-site timeouts
	timeoutHandler := http.TimeoutHandler(
		proxy,
		Max(cfg.Timeouts.Read, cfg.Timeouts.Write),
		"Request timeout",
	)

	// Add request/response logging
	loggedHandler := loggingHandler(logger, timeoutHandler)

	return &Handler{
		Site:    cfg,
		Handler: loggedHandler,
		Logger:  logger,
		lb:      lb,
	}, nil
}
