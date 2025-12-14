package main

import (
	"log"
	"net/http"
	"reverse-proxy/internal/application/config"
	"reverse-proxy/internal/application/host"
	"reverse-proxy/internal/application/site"
	"reverse-proxy/internal/models/global"
)

func main() {
	settings, err := config.LoadSettings("./config/settings.yml")
	if err != nil {
		log.Fatalf("Failed to load settings: %s", err)
	}

	sites, err := config.LoadConfigs("./config")
	if err != nil {
		log.Fatalf("Error loading sites: %v", err)
	}

	configs := make(map[string]http.Handler)
	for domain, cfg := range sites {
		handler := site.NewSiteHandler(cfg)
		configs[domain] = handler.Handler
	}

	router := host.HostRouter(configs)
	go func() {
		defaultServer(settings, router)
	}()
	go func() {
		tlsServer(settings, router)
	}()
	log.Printf("starting app")
}

func defaultServer(settings *global.Settings, router http.Handler) {
	server := &http.Server{
		Addr:           settings.Server.Listen,
		ReadTimeout:    settings.Server.Timeouts.Read,
		WriteTimeout:   settings.Server.Timeouts.Write,
		IdleTimeout:    settings.Server.Timeouts.Idle,
		MaxHeaderBytes: settings.Server.Limits.MaxHeaderBytes,
		Handler:        router,
	}

	log.Printf("Listening on %s\n", settings.Server.Listen)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}

func tlsServer(settings *global.Settings, router http.Handler) {
	if settings.Server.TLS != nil {
		tlsCfg := settings.Server.TLS

		if tlsCfg.CertFile == "" || tlsCfg.KeyFile == "" {
			log.Fatal("TLS enabled but cert_file or key_file missing")
		}

		httpsServer := &http.Server{
			Addr:         tlsCfg.Listen,
			Handler:      router,
			ReadTimeout:  settings.Server.Timeouts.Read,
			WriteTimeout: settings.Server.Timeouts.Write,
			IdleTimeout:  settings.Server.Timeouts.Idle,
		}

		log.Printf("HTTPS listening on %s\n", tlsCfg.Listen)
		log.Fatal(httpsServer.ListenAndServeTLS(
			tlsCfg.CertFile,
			tlsCfg.KeyFile,
		))
	}
}
