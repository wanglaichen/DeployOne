package main

import (
	"flag"
	"log"
	"net/http"

	"webdown/internal/app"
	"webdown/internal/config"
	"webdown/internal/store"
)

func main() {
	configPath := flag.String("config", "config/config.json", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	st, err := store.New(cfg.DataFile)
	if err != nil {
		log.Fatalf("init store: %v", err)
	}

	server, err := app.New(cfg, st)
	if err != nil {
		log.Fatalf("init app: %v", err)
	}

	log.Printf("webdown listening on http://%s", cfg.Addr())
	log.Printf("public download base url: %s", cfg.PublicBaseURL)
	if err := http.ListenAndServe(cfg.Addr(), server.Handler()); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
