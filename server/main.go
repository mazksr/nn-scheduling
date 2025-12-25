package main

import (
	"log"
	"net/http"
)

func main() {
	initEnv()

	cfg := initConfig()

	mux := http.NewServeMux()

	mux.HandleFunc("POST /infer-rr", cfg.roundRobinHandler)
	mux.HandleFunc("POST /infer-single", cfg.singleWorkerHandler)
	mux.HandleFunc("POST /infer-leastconn", cfg.leastConnHandler)
	mux.HandleFunc("POST /reg-worker", cfg.inferenceRegHandler)
	mux.HandleFunc("GET /health", healthHandler)

	server := &http.Server {
		Handler: mux,
		Addr: ":8080",
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v\n", err)
	}
}
