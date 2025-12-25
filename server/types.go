package main

import (
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
	cfg.workers = append(cfg.workers, worker {
		url: "http://"+addr+":"+workerPort,
		})
}

func (cfg *apiConfig) getLeastConnection() int {
	cfg.mu.RLock()
	defer cfg.mu.RUnlock()
	least := cfg.workers[0].connCount.Load()
	leastPos := 0
	for i := 1; i < len(cfg.workers); i++ {
		val := cfg.workers[i].connCount.Load()
		if val < least {
			least = val
			leastPos = i
		}
	}

	return leastPos
}

func (cfg *apiConfig) getRoundRobin() int {
	i := cfg.roundRobinWorker.Add(1) - 1
	return int(i % uint64(len(cfg.workers)))
}
