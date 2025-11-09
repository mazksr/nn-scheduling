package main

import (
	"testing"
)

func TestGetAndNext(t *testing.T) {
	cfg := &apiConfig{
		numWorkers:      3,
	}

	// Expected sequence: 0,1,2,0,1,2,...
	want := []int{0, 1, 2, 0, 1, 2}
	got := make([]int, len(want))

	for i := range got {
		got[i] = cfg.getRoundRobin()
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("3-worker case: at index %d: got %d, want %d", i, got[i], want[i])
		}
	}

	// --- Additional case: 4 workers ---
	cfg = &apiConfig{
		numWorkers:      4,
	}

	want = []int{0, 1, 2, 3, 0, 1, 2, 3}
	got = make([]int, len(want))

	for i := range got {
		got[i] = cfg.getRoundRobin()
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("4-worker case: at index %d: got %d, want %d", i, got[i], want[i])
		}
	}
}

func TestInitializeWorkersAssignsURLs(t *testing.T) {
	originalWorkerURL := workerURL
	workerURL = "http://example.com"
	t.Cleanup(func() { workerURL = originalWorkerURL })

	cfg := &apiConfig{
		numWorkers: 3,
		workers:    make([]worker, 3),
	}

	cfg.initializeWorkers()

	want := []string{
		"http://example.com:8000/infer",
		"http://example.com:8001/infer",
		"http://example.com:8002/infer",
	}

	for i := range want {
		if cfg.workers[i].url != want[i] {
			t.Fatalf("worker %d url mismatch: got %q, want %q", i, cfg.workers[i].url, want[i])
		}
	}
}

func TestInitializeWorkersLeavesExtraEntriesUntouched(t *testing.T) {
	originalWorkerURL := workerURL
	workerURL = "http://internal"
	t.Cleanup(func() { workerURL = originalWorkerURL })

	cfg := &apiConfig{
		numWorkers: 2,
		workers: []worker{
			{url: "old-0"},
			{url: "old-1"},
			{url: "keep-me"},
		},
	}

	cfg.initializeWorkers()

	want := []string{
		"http://internal:8000/infer",
		"http://internal:8001/infer",
	}

	for i := range want {
		if cfg.workers[i].url != want[i] {
			t.Fatalf("worker %d url mismatch: got %q, want %q", i, cfg.workers[i].url, want[i])
		}
	}

	if cfg.workers[2].url != "keep-me" {
		t.Fatalf("expected extra worker entries beyond numWorkers to remain untouched, got %q", cfg.workers[2].url)
	}
}
