package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync/atomic"

	"github.com/joho/godotenv"
)

var workerURL string
var numWorkers int

type inferStruct struct {
	file []byte
}

type apiConfig struct {
	numWorkers int
	workers []worker
	roundRobinWorker atomic.Uint64
}

type worker struct {
	url string
	connCount atomic.Uint32
}

func (cfg *apiConfig) initializeWorkers() {
	for i := range cfg.numWorkers {
		cfg.workers[i].url = workerURL + ":800" + strconv.Itoa(i) + "/infer"
	}
}

func main() {
	err := godotenv.Load()
	if err != nil {fmt.Fprintf(os.Stderr, "Error loading environment: %v\n", err); os.Exit(1)}

	mux := http.NewServeMux()

	workerURL = os.Getenv("WORKER_URL")
	workerEnv, err := strconv.Atoi(os.Getenv("NUM_WORKERS"))
	if err != nil || numWorkers < 1{
		fmt.Println("error loading NUM_WORKERS")
		return
	}
	numWorkers = workerEnv

	cfg := &apiConfig{
		workers: make([]worker, numWorkers),
		numWorkers: numWorkers,
	}
	cfg.initializeWorkers()

	mux.HandleFunc("POST /infer-rr", cfg.roundRobinHandler)
	mux.HandleFunc("POST /infer-single", cfg.singleWorkerHandler)
	mux.HandleFunc("POST /infer-", cfg.singleWorkerHandler)
	mux.HandleFunc("GET /health", healthHandler)

	server := &http.Server {
		Handler: mux,
		Addr: ":8080",
	}
	server.ListenAndServe()
}

func (cfg *apiConfig) getRoundRobin() int {
	i := cfg.roundRobinWorker.Add(1) - 1
	return int(i % uint64(cfg.numWorkers))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {}

func (cfg *apiConfig) singleWorkerHandler(w http.ResponseWriter, r *http.Request) {
	forward(w, r, cfg.workers[0].url)
}

func (cfg *apiConfig) roundRobinHandler(w http.ResponseWriter, r *http.Request) {
	serverIndex := cfg.getRoundRobin()
	forward(w, r, cfg.workers[serverIndex].url)
}

func forward(w http.ResponseWriter, r *http.Request, url string) {
	request, err := http.NewRequest(r.Method, url, r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte{})
		return
	}
	request.Header = r.Header.Clone()
	request.Host = ""

	client := &http.Client{}
	res, err := client.Do(request)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte{})
		return
	}
	defer res.Body.Close()

	for key, val := range res.Header {
		for _, v := range val {
			w.Header().Add(key, v)
		}
	}

	w.WriteHeader(res.StatusCode)
	io.Copy(w, res.Body)
}
