package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"
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

func (cfg *apiConfig) p2cHandler(w http.ResponseWriter, r *http.Request) {
	wk, err := cfg.getPowerOfTwoChoices()
	if err != nil {
		http.Error(w, "No workers available", http.StatusServiceUnavailable)
		return
	}

	// Increment before the request; decrement when handler exits.
	wk.connCount.Add(1)
	defer wk.connCount.Add(-1)

	w.Header().Set("X-Worker-Id", cfg.getWorkerID(wk.url))

	res, err := cfg.makeInference(r, wk.url)
	if err != nil {
		log.Printf("p2cHandler: inference error: %v\n", err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
	defer res.Body.Close()

	if err = sendResult(w, res); err != nil {
		log.Printf("p2cHandler: sendResult error: %v\n", err)
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
	w.Write([]byte(" ")) // Force chunked encoding so the client blocks
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

func (cfg *apiConfig) hybridPayloadHandler(w http.ResponseWriter, r *http.Request) {
	payloadSizeMB := float64(r.ContentLength) / (1024.0 * 1024.0)
	if payloadSizeMB <= 0 {
		payloadSizeMB = 1.0 // fallback
	}

	wk, expectedLoad, err := cfg.getHybridPayloadWorker(payloadSizeMB)
	if err != nil {
		http.Error(w, "No workers available", http.StatusServiceUnavailable)
		return
	}

	wk.connCount.Add(1)
	defer wk.connCount.Add(-1)

	workerID := cfg.getWorkerID(wk.url)
	w.Header().Set("X-Worker-Id", workerID)

	startTime := time.Now()
	res, err := cfg.makeInference(r, wk.url)
	
	actualDurationMs := float64(time.Since(startTime).Milliseconds())

	// Beta: static GPU time
	beta := 8.0

	wk.mu.Lock()
	wk.currentWeight -= expectedLoad
	
	// The Negative Alpha Trap Guardrail: Only update alpha on successful responses
	// and when actual processing time exceeds static GPU time.
	if err == nil && res != nil && res.StatusCode == http.StatusOK {
		if actualDurationMs > beta {
			alphaMeasured := (actualDurationMs - beta) / payloadSizeMB
			wk.alpha = (0.8 * wk.alpha) + (0.2 * alphaMeasured)
			fmt.Printf("Worker %s new Alpha: %.2f\n", workerID, wk.alpha)
		}
	}
	wk.mu.Unlock()

	if err != nil {
		log.Printf("hybridPayloadHandler: inference error: %v\n", err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
	defer res.Body.Close()

	if err = sendResult(w, res); err != nil {
		log.Printf("hybridPayloadHandler: sendResult error: %v\n", err)
	}
}
