package main

import (
	"log/slog"
	"net/http"
	"os"
	"reverse-proxy/internal/application/config"
	"reverse-proxy/internal/application/host"
	"reverse-proxy/internal/application/site"
	"reverse-proxy/internal/models/global"
	"strings"
	"sync"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	settings, err := config.LoadSettings("./config/settings.yml")
	if err != nil {
		logger.Error("Failed to load settings", "error", err)
		os.Exit(1)
	}

	sites, err := config.LoadConfigs("./config")
	if err != nil {
		logger.Error("Error loading sites", "error", err)
		os.Exit(1)
	}

	configs := make(map[string]http.Handler)
	for domain, cfg := range sites {
		handler, err := site.NewSiteHandler(logger, cfg)
		if err != nil {
			logger.Error("Failed to create handler", "domain", domain, "error", err)
			continue
		}
		configs[domain] = handler.Handler
	}

	router := host.Router(configs)

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		defaultServer(logger, settings, router)
	}()

	if settings.Server.TLS != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tlsServer(logger, settings, router)
		}()
	}

	logger.Info("starting app")
	wg.Wait()
}

func defaultServer(logger *slog.Logger, settings *global.Settings, router http.Handler) {
	handler := router
	if settings.Server.TLS != nil && settings.Server.TLS.RedirectHTTP {
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			host := r.Host
			if strings.Contains(host, ":") {
				host = strings.Split(host, ":")[0]
			}
			target := "https://" + host + r.RequestURI
			logger.Info("Redirecting to HTTPS", "from", r.Host+r.RequestURI, "to", target)
			http.Redirect(w, r, target, http.StatusMovedPermanently)
		})
	}

	server := &http.Server{
		Addr:           settings.Server.Listen,
		ReadTimeout:    settings.Server.Timeouts.Read,
		WriteTimeout:   settings.Server.Timeouts.Write,
		IdleTimeout:    settings.Server.Timeouts.Idle,
		MaxHeaderBytes: settings.Server.Limits.MaxHeaderBytes,
		Handler:        handler,
	}

	logger.Info("Listening on", "addr", settings.Server.Listen)
	if err := server.ListenAndServe(); err != nil {
		logger.Error("Error starting server", "error", err)
		os.Exit(1)
	}
}

func tlsServer(logger *slog.Logger, settings *global.Settings, router http.Handler) {
	if settings.Server.TLS != nil {
		tlsCfg := settings.Server.TLS

		if tlsCfg.CertFile == "" || tlsCfg.KeyFile == "" {
			logger.Error("TLS enabled but cert_file or key_file missing")
			os.Exit(1)
		}

		httpsServer := &http.Server{
			Addr:         tlsCfg.Listen,
			Handler:      router,
			ReadTimeout:  settings.Server.Timeouts.Read,
			WriteTimeout: settings.Server.Timeouts.Write,
			IdleTimeout:  settings.Server.Timeouts.Idle,
		}

		logger.Info("HTTPS listening on", "addr", tlsCfg.Listen)
		if err := httpsServer.ListenAndServeTLS(
			tlsCfg.CertFile,
			tlsCfg.KeyFile,
		); err != nil {
			logger.Error("Error starting HTTPS server", "error", err)
			os.Exit(1)
		}
	}
}
