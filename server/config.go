package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
)

func initConfig() *apiConfig {
	return &apiConfig{
		workers:    make([]*worker, 0),
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func initEnv(cfg *apiConfig) {
	// Attempt to load .env file if present, but do not fail if it's missing (e.g., in Docker production).
	_ = godotenv.Load()

	workerPort := os.Getenv("WORKERS_PORT")
	if workerPort == "" {
		log.Fatalln("error: WORKERS_PORT environment variable is not set (must be set in environment or .env)")
	}
	cfg.workerPort = workerPort

	secret := os.Getenv("REGISTRATION_SECRET")
	if secret == "" {
		log.Fatalln("error: REGISTRATION_SECRET environment variable is not set (must be set in environment or .env)")
	}
	cfg.registrationSecret = secret
}
