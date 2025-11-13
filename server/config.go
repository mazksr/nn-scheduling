package main

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

var workerURL string // Not mutated after
var numWorkers int	// loaded from environment

func initConfig() *apiConfig {
	cfg := &apiConfig{
		workers: make([]worker, numWorkers), // numWorkers -> ./config.go
	}
	cfg.initializeWorkers()

	return cfg
}

func initEnv() {
	err := godotenv.Load()
	if err != nil {log.Fatalf("Error loading environment: %v\n", err)}

	workerURL = os.Getenv("WORKER_URL")
	if workerURL == "" {
		log.Fatalln("error loading WORKER_URL")
	}
	workerEnv, err := strconv.Atoi(os.Getenv("NUM_WORKERS"))
	if err != nil || workerEnv < 1{
		log.Fatalln("error loading NUM_WORKERS")
	}
	numWorkers = workerEnv
}
