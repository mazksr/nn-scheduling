package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
)

var workerURL string

type apiConfig struct {
	numWorkers int
	Servers []string
	roundRobinState int
	mu *sync.Mutex
}

func (cfg *apiConfig) initializeWorkers() {
	for i := range cfg.numWorkers {
		cfg.Servers[i] = workerURL + ":800" + strconv.Itoa(i) + "/infer"
	}
}

func main() {
	mux := http.NewServeMux()

	workerURL = os.Getenv("WORKER_URL")
	numWorkers, err := strconv.Atoi(os.Getenv("NUM_WORKERS"))
	if err != nil {
		fmt.Println("error loading NUM_WORKERS")
		return
	}

	cfg := &apiConfig{
		Servers: make([]string, numWorkers),
		mu: &sync.Mutex{},
		numWorkers: numWorkers,
	}
	cfg.initializeWorkers()

	mux.HandleFunc("POST /infer", cfg.roundRobinHandler)

	server := &http.Server {
		Handler: mux,
		Addr: ":8080",
	}
	server.ListenAndServe()
}

func (cfg *apiConfig) roundRobinHandler(w http.ResponseWriter, r *http.Request) {
	cfg.mu.Lock()
	serverIndex := cfg.roundRobinState
	if serverIndex >= cfg.numWorkers-1 {
		cfg.roundRobinState = -1
	}
	cfg.roundRobinState += 1
	cfg.mu.Unlock()

	fmt.Println(cfg.Servers[serverIndex])

}
