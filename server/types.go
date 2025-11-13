package main

import (
	"strconv"
	"sync"
	"sync/atomic"
)

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

func (cfg *apiConfig) getRoundRobin() int {
	i := cfg.roundRobinWorker.Add(1) - 1
	return int(i % uint64(numWorkers))
}
