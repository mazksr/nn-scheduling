package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/joho/godotenv"
)

var workerURL string // Not mutated after
var numWorkers int	// loaded from environment

type apiConfig struct {
	workers []worker
	roundRobinWorker atomic.Uint64
	mu sync.Mutex
}

type worker struct {
	url string
	connCount atomic.Int32
}

func (cfg *apiConfig) initializeWorkers() {
	for i := range numWorkers {
		cfg.workers[i].url = workerURL + ":800" + strconv.Itoa(i) + "/infer"
	}
}

func main() {
	err := godotenv.Load()
	if err != nil {log.Fatalf("Error loading environment: %v\n", err)}

	mux := http.NewServeMux()

	workerURL = os.Getenv("WORKER_URL")
	if workerURL == "" {
		log.Fatalln("error loading WORKER_URL")
	}
	workerEnv, err := strconv.Atoi(os.Getenv("NUM_WORKERS"))
	if err != nil || workerEnv < 1{
		log.Fatalln("error loading NUM_WORKERS")
	}
	numWorkers = workerEnv

	cfg := &apiConfig{
		workers: make([]worker, numWorkers),
	}
	cfg.initializeWorkers()

	mux.HandleFunc("POST /infer-rr", cfg.roundRobinHandler)
	mux.HandleFunc("POST /infer-single", cfg.singleWorkerHandler)
	mux.HandleFunc("POST /infer-leastconn", cfg.leastConnHandler)
	mux.HandleFunc("GET /health", healthHandler)

	server := &http.Server {
		Handler: mux,
		Addr: ":8080",
	}
	server.ListenAndServe()
}

func (cfg *apiConfig) getLeastConnection() int {
	least := cfg.workers[0].connCount.Load()
	leastPos := 0
	for i := 1; i < numWorkers; i++ {
		val := cfg.workers[i].connCount.Load()
		if val < least {
			least = val
			leastPos = i
		}
	}

	return leastPos
}

func (cfg *apiConfig) leastConnHandler(w http.ResponseWriter, r *http.Request) {
	least := cfg.getLeastConnection()

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

func (cfg *apiConfig) getRoundRobin() int {
	i := cfg.roundRobinWorker.Add(1) - 1
	return int(i % uint64(numWorkers))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {}

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

func makeInference(r *http.Request, url string) (*http.Response, error) {
	request, err := http.NewRequest(r.Method, url, r.Body)
	if err != nil {
		return &http.Response{}, err
	}
	request.Header = r.Header.Clone()
	request.Host = ""

	client := &http.Client{}
	return client.Do(request)
}

func sendResult(w http.ResponseWriter, res *http.Response) error {
	for key, val := range res.Header {
		for _, v := range val {
			w.Header().Add(key, v)
		}
	}

	w.WriteHeader(res.StatusCode)
	_, err := io.Copy(w, res.Body)

	return err
}

func forward(w http.ResponseWriter, r *http.Request, url string) error {
	res, err := makeInference(r, url)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	err = sendResult(w, res)

	return err
}
