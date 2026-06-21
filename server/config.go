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
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading environment: %v\n", err)
	}

	workerPort := os.Getenv("WORKERS_PORT")
	if workerPort == "" {
		log.Fatalln("error loading WORKERS_PORT from .env")
	}
	cfg.workerPort = workerPort

	secret := os.Getenv("REGISTRATION_SECRET")
	if secret == "" {
		log.Fatalln("error loading REGISTRATION_SECRET from .env")
	}
	cfg.registrationSecret = secret
}
