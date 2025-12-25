package main

import (
	"net/http"
	"strings"
)

func (cfg *apiConfig) singleWorkerHandler(w http.ResponseWriter, r *http.Request) {
	err := forward(w, r, cfg.workers[0].url)
	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
	}
}

func (cfg *apiConfig) roundRobinHandler(w http.ResponseWriter, r *http.Request) {
	serverIndex := cfg.getRoundRobin()
	err := forward(w, r, cfg.workers[serverIndex].url)
	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
	}
}

func (cfg *apiConfig) leastConnHandler(w http.ResponseWriter, r *http.Request) {
	least := cfg.getLeastConnection()
	// log.Println(least)

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

func (cfg *apiConfig) inferenceRegHandler(_ http.ResponseWriter, r *http.Request) {
	clientAddress := strings.Split(r.RemoteAddr, ":")[0]
	cfg.registerWorker(clientAddress)
}


func healthHandler(w http.ResponseWriter, r *http.Request) {}
