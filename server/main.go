package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/joho/godotenv"
)

var workerURL string

type inferStruct struct {
	file []byte
}

type apiConfig struct {
	numWorkers int
	Servers []string
	roundRobinState int
	mu sync.Mutex
}

func (cfg *apiConfig) initializeWorkers() {
	for i := range cfg.numWorkers {
		cfg.Servers[i] = workerURL + ":800" + strconv.Itoa(i) + "/infer"
	}
}

func main() {
	err := godotenv.Load()
	if err != nil {fmt.Fprintf(os.Stderr, "Error loading environment: %v\n", err); os.Exit(1)}

	mux := http.NewServeMux()

	workerURL = os.Getenv("WORKER_URL")
	numWorkers, err := strconv.Atoi(os.Getenv("NUM_WORKERS"))
	if err != nil || numWorkers < 1{
		fmt.Println("error loading NUM_WORKERS")
		return
	}

	cfg := &apiConfig{
		Servers: make([]string, numWorkers),
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

func (cfg *apiConfig) getRoundRobin() int {
	var serverIndex int

	if cfg.numWorkers > 1 {
		cfg.mu.Lock()
		serverIndex = cfg.roundRobinState
		if serverIndex >= cfg.numWorkers-1 {
			cfg.roundRobinState = 0
		} else {
			cfg.roundRobinState += 1
		}
		cfg.mu.Unlock()
	} else {
		serverIndex = 0
	}

	return serverIndex
}

func (cfg *apiConfig) roundRobinHandler(w http.ResponseWriter, r *http.Request) {
	serverIndex := cfg.getRoundRobin()
	forward(w, r, cfg.Servers[serverIndex])
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
