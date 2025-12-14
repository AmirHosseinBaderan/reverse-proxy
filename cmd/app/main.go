package main

import (
	"log"
	"net/http"
	"reverse-proxy/internal/application/config"
	"reverse-proxy/internal/application/host"
	"reverse-proxy/internal/application/site"
)

func main() {
	settings, err := config.LoadSettings("./config/settings.yml")
	if err != nil {
		log.Fatalf("Failed to load settings: %s", err)
	}

	sites, err := config.LoadConfigs("./config")
	if err != nil {
		log.Fatalf("Error loading settings: %v", err)
	}

	configs := make(map[string]http.Handler)
	for domain, cfg := range sites {
		handler := site.NewSiteHandler(cfg)
		configs[domain] = handler.Handler
	}

	router := host.HostRouter(configs)

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
