package main

import (
	"errors"
	"sync"
	"sync/atomic"
)

type apiConfig struct {
	workers []worker
	roundRobinWorker atomic.Uint64
	mu sync.RWMutex
}

type worker struct {
	url string
	connCount atomic.Int32
}

func (cfg *apiConfig) registerWorker(addr string) {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()

	url := "http://"+addr+":"+workerPort
	for _, w := range cfg.workers {
		if w.url == url {
			return
		}
	}

	cfg.workers = append(cfg.workers, worker {
		url: url,
		})
}

func (cfg *apiConfig) getSingleWorker() (string, error) {
	cfg.mu.RLock()
	defer cfg.mu.RUnlock()

	if len(cfg.workers) == 0 {
		return "", errors.New("no workers available")
	}
	return cfg.workers[0].url, nil
}

func (cfg *apiConfig) getLeastConnection() (int, error) {
	cfg.mu.RLock()
	defer cfg.mu.RUnlock()

	if len(cfg.workers) == 0 {
		return 0, errors.New("no workers available")
	}

	least := cfg.workers[0].connCount.Load()
	leastPos := 0
	for i := 1; i < len(cfg.workers); i++ {
		val := cfg.workers[i].connCount.Load()
		if val < least {
			least = val
			leastPos = i
		}
	}

	return leastPos, nil
}

func (cfg *apiConfig) getRoundRobin() (int, error) {
	cfg.mu.RLock()
	defer cfg.mu.RUnlock()

	if len(cfg.workers) == 0 {
		return 0, errors.New("no workers available")
	}

	i := cfg.roundRobinWorker.Add(1) - 1
	return int(i % uint64(len(cfg.workers))), nil
}
