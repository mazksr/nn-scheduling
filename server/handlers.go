package main

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
)

func (cfg *apiConfig) singleWorkerHandler(w http.ResponseWriter, r *http.Request) {
	wk, err := cfg.getSingleWorker()
	if err != nil {
		http.Error(w, "No workers available", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("X-Worker-Id", cfg.getWorkerID(wk.url))
	if err = cfg.forward(w, r, wk.url); err != nil {
		log.Printf("singleWorkerHandler: forward error: %v\n", err)
	}
}

func (cfg *apiConfig) roundRobinHandler(w http.ResponseWriter, r *http.Request) {
	wk, err := cfg.getRoundRobin()
	if err != nil {
		http.Error(w, "No workers available", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("X-Worker-Id", cfg.getWorkerID(wk.url))
	if err = cfg.forward(w, r, wk.url); err != nil {
		log.Printf("roundRobinHandler: forward error: %v\n", err)
	}
}

func (cfg *apiConfig) leastConnHandler(w http.ResponseWriter, r *http.Request) {
	wk, err := cfg.getLeastConnection()
	if err != nil {
		http.Error(w, "No workers available", http.StatusServiceUnavailable)
		return
	}

	// Increment before the request; decrement when handler exits.
	// Safe even if the worker is evicted mid-flight because wk is a pointer.
	wk.connCount.Add(1)
	defer wk.connCount.Add(-1)

	w.Header().Set("X-Worker-Id", cfg.getWorkerID(wk.url))

	res, err := cfg.makeInference(r, wk.url)
	if err != nil {
		log.Printf("leastConnHandler: inference error: %v\n", err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
	defer res.Body.Close()

	if err = sendResult(w, res); err != nil {
		log.Printf("leastConnHandler: sendResult error: %v\n", err)
	}
}

func (cfg *apiConfig) inferenceRegHandler(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Registration-Secret") != cfg.registrationSecret {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	clientAddress, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		clientAddress = r.RemoteAddr
	}
	wk := cfg.registerWorker(clientAddress)
	log.Printf("[+] Registered worker from: %s\n", clientAddress)
	
	w.WriteHeader(http.StatusOK)
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	select {
	case <-wk.evicted:
		// Worker was evicted by the scheduler
		log.Printf("[-] Worker %s evicted, closing registration connection.\n", clientAddress)
	case <-r.Context().Done():
		// Discovery agent disconnected
		log.Printf("[-] Discovery agent for %s disconnected, evicting worker.\n", clientAddress)
		cfg.removeWorker(wk.url)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
