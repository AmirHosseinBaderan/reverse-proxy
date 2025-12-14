package site

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"reverse-proxy/internal/models/global"
	"time"
)

type Handler struct {
	site    *global.SiteConfig
	handler http.Handler
}

func NewSiteHandler(cfg *global.SiteConfig) *Handler {
	target, err := url.Parse(cfg.Proxy.Upstream)
	if err != nil {
		panic("invalid upstream for " + cfg.Domain)
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
		return nil
	}

	// error handler
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, "Upstream error", http.StatusBadGateway)
	}

	// Apply per-site timeouts
	handler := http.TimeoutHandler(
		proxy,
		max(cfg.Timeouts.Read, cfg.Timeouts.Write),
		"Request timeout",
	)

	return &Handler{
		site:    cfg,
		handler: handler,
	}
}
