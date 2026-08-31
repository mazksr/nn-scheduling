package main

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type apiConfig struct {
	workers            []*worker
	roundRobinWorker   atomic.Uint64
	mu                 sync.RWMutex
	httpClient         *http.Client
	workerPort         string
	registrationSecret string
}

// worker represents a single backend inference node.
// Stored as a pointer so that in-flight requests can safely hold a reference
// even after the worker has been evicted from the pool.
type worker struct {
	url           string
	connCount     atomic.Int32
	evicted       chan struct{}
	alpha         float64      // The dynamic CPU processing coefficient
	currentWeight float64      // Wi: The current expected queue load
	mu            sync.RWMutex // Prevents data races during concurrent traffic
}

func (cfg *apiConfig) registerWorker(addr string) *worker {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()

	url := fmt.Sprintf("http://%s:%s", addr, cfg.workerPort)
	for _, w := range cfg.workers {
		if w.url == url {
			return w
		}
	}

	w := &worker{
		url:           url,
		evicted:       make(chan struct{}),
		alpha:         25.5, // Initialized with profiled alpha
		currentWeight: 0.0,
	}
	cfg.workers = append(cfg.workers, w)
	return w
}

func (cfg *apiConfig) removeWorker(url string) {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()

	for i, w := range cfg.workers {
		if w.url == url {
			close(w.evicted)
			cfg.workers = append(cfg.workers[:i], cfg.workers[i+1:]...)
			log.Printf("[-] Removed dead worker: %s\n", url)
			return
		}
	}
}

// getSingleWorker always returns the first (pinned) worker.
// Useful when all requests must go to one known-good node.
func (cfg *apiConfig) getSingleWorker() (*worker, error) {
	cfg.mu.RLock()
	defer cfg.mu.RUnlock()

	if len(cfg.workers) == 0 {
		return nil, errors.New("no workers available")
	}
	return cfg.workers[0], nil
}

func (cfg *apiConfig) getLeastConnection() (*worker, error) {
	cfg.mu.RLock()
	defer cfg.mu.RUnlock()

	if len(cfg.workers) == 0 {
		return nil, errors.New("no workers available")
	}

	best := cfg.workers[0]
	for _, w := range cfg.workers[1:] {
		if w.connCount.Load() < best.connCount.Load() {
			best = w
		}
	}

	return best, nil
}

func (cfg *apiConfig) getRoundRobin() (*worker, error) {
	cfg.mu.RLock()
	defer cfg.mu.RUnlock()

	if len(cfg.workers) == 0 {
		return nil, errors.New("no workers available")
	}

	// The counter wraps naturally at 2^64; harmless in practice.
	i := cfg.roundRobinWorker.Add(1) - 1
	return cfg.workers[int(i%uint64(len(cfg.workers)))], nil
}

func (cfg *apiConfig) getPowerOfTwoChoices() (*worker, error) {
	cfg.mu.RLock()
	defer cfg.mu.RUnlock()

	n := len(cfg.workers)
	if n == 0 {
		return nil, errors.New("no workers available")
	}
	if n == 1 {
		return cfg.workers[0], nil
	}

	i := rand.Intn(n)
	j := rand.Intn(n - 1)
	if j >= i {
		j++
	}

	w1 := cfg.workers[i]
	w2 := cfg.workers[j]

	if w1.connCount.Load() <= w2.connCount.Load() {
		return w1, nil
	}
	return w2, nil
}

// startHealthChecks spawns a background goroutine that pings every registered
// worker's /health endpoint at the given interval and evicts any that fail.
func (cfg *apiConfig) startHealthChecks(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			cfg.mu.RLock()
			urls := make([]string, len(cfg.workers))
			for i, w := range cfg.workers {
				urls[i] = w.url
			}
			cfg.mu.RUnlock()

			for _, url := range urls {
				resp, err := cfg.httpClient.Get(url + "/health")
				if err != nil || resp.StatusCode != http.StatusOK {
					cfg.removeWorker(url)
					continue
				}
				resp.Body.Close()
			}
		}
	}()
}

func (cfg *apiConfig) getHybridPayloadWorker(payloadSizeMB float64) (*worker, float64, error) {
	cfg.mu.RLock()
	defer cfg.mu.RUnlock()

	if len(cfg.workers) == 0 {
		return nil, 0, errors.New("no workers available")
	}

	var bestWorker *worker
	var minSi float64
	var expectedLoad float64

	for i, w := range cfg.workers {
		w.mu.RLock()
		alpha := w.alpha
		weight := w.currentWeight
		w.mu.RUnlock()

		load := alpha * payloadSizeMB
		si := weight + load

		if i == 0 || si < minSi {
			bestWorker = w
			minSi = si
			expectedLoad = load
		}
	}

	bestWorker.mu.Lock()
	bestWorker.currentWeight += expectedLoad
	bestWorker.mu.Unlock()

	return bestWorker, expectedLoad, nil
}

