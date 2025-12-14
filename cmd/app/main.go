package main

import (
	"log"
	"net/http"
	"reverse-proxy/internal/application/config"
)

func main() {
	settings, err := config.LoadSettings("./config/settings.yml")
	if err != nil {
		log.Fatalf("Failed to load settings: %s", err)
	}

	server := &http.Server{
		Addr:           settings.Server.Listen,
		ReadTimeout:    settings.Server.Timeouts.Read,
		WriteTimeout:   settings.Server.Timeouts.Write,
		IdleTimeout:    settings.Server.Timeouts.Idle,
		MaxHeaderBytes: settings.Server.Limits.MaxHeaderBytes,
	}

	sites, err := config.LoadSettings("./config")
	if err != nil {
		log.Fatalf("Error loading settings: %v", err)
	}

	log.Printf("Listening on %s\n", settings.Server.Listen)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
