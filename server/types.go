package main

import (
	"errors"
	"fmt"
	"log"
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
	url       string
	connCount atomic.Int32
}

func (cfg *apiConfig) registerWorker(addr string) {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()

	url := fmt.Sprintf("http://%s:%s", addr, cfg.workerPort)
	for _, w := range cfg.workers {
		if w.url == url {
			return
		}
	}

	cfg.workers = append(cfg.workers, &worker{url: url})
}

func (cfg *apiConfig) removeWorker(url string) {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()

	for i, w := range cfg.workers {
		if w.url == url {
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
