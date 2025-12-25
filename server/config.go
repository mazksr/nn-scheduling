package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

var workerPort string

func initConfig() *apiConfig {
	cfg := &apiConfig{
		workers: make([]worker, 4),
	}

	return cfg
}

func initEnv() {
	err := godotenv.Load()
	if err != nil {log.Fatalf("Error loading environment: %v\n", err)}

	workerPortEnv := os.Getenv("WORKERS_PORT")
	if workerPortEnv == "" {
		log.Fatalln("error loading WORKERS_PORT from .env")
	}
	workerPort = workerPortEnv
}
