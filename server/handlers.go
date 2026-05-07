package main

import (
	"net/http"
	"strings"
)

func (cfg *apiConfig) singleWorkerHandler(w http.ResponseWriter, r *http.Request) {
	url, err := cfg.getSingleWorker()
	if err != nil {
		http.Error(w, "No workers available", http.StatusServiceUnavailable)
		return
	}
	err = forward(w, r, url)
	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
	}
}

func (cfg *apiConfig) roundRobinHandler(w http.ResponseWriter, r *http.Request) {
	serverIndex, err := cfg.getRoundRobin()
	if err != nil {
		http.Error(w, "No workers available", http.StatusServiceUnavailable)
		return
	}
	err = forward(w, r, cfg.workers[serverIndex].url)
	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
	}
}

func (cfg *apiConfig) leastConnHandler(w http.ResponseWriter, r *http.Request) {
	least, err := cfg.getLeastConnection()
	if err != nil {
		http.Error(w, "No workers available", http.StatusServiceUnavailable)
		return
	}

	cfg.workers[least].connCount.Add(1)
	res, err := makeInference(r, cfg.workers[least].url)
	defer cfg.workers[least].connCount.Add(-1)
	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
	defer res.Body.Close()

	err = sendResult(w, res)
	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
	}
}

func (cfg *apiConfig) inferenceRegHandler(w http.ResponseWriter, r *http.Request) {
	clientAddress := strings.Split(r.RemoteAddr, ":")[0]
	cfg.registerWorker(clientAddress)
	w.WriteHeader(http.StatusOK)
}


func healthHandler(w http.ResponseWriter, r *http.Request) {}
