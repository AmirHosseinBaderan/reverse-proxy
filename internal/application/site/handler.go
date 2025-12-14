package site

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"reverse-proxy/internal/models/global"
	"time"
)

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
}

func NewSiteHandler(logger *slog.Logger, cfg *global.SiteConfig) (*Handler, error) {
	target, err := url.Parse(cfg.Proxy.Upstream)
	if err != nil {
		return nil, fmt.Errorf("invalid upstream for %s: %w", cfg.Domain, err)
	}

	transport := &http.Transport{
		MaxIdleConns:        1000,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = transport

	// Add / override request headers
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		for k, v := range cfg.Proxy.Headers {
			req.Header.Set(k, v)
		}
	}

	// Response header hook
	proxy.ModifyResponse = func(resp *http.Response) error {
		logger.Info("Upstream response", "status", resp.StatusCode, "size", resp.ContentLength, "url", resp.Request.URL.String())
		return nil
	}

	// error handler
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		logger.Error("Upstream error", "domain", cfg.Domain, "url", r.URL.String(), "error", err)
		http.Error(w, "Upstream error", http.StatusBadGateway)
	}

	// Apply per-site timeouts
	timeoutHandler := http.TimeoutHandler(
		proxy,
		max(cfg.Timeouts.Read, cfg.Timeouts.Write),
		"Request timeout",
	)

	// Add request/response logging
	loggedHandler := loggingHandler(logger, timeoutHandler)

	return &Handler{
		Site:    cfg,
		Handler: loggedHandler,
		Logger:  logger,
	}, nil
}
