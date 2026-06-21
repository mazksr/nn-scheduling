package main

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

// mkConfig builds an apiConfig populated with the given URLs.
func mkConfig(urls []string) *apiConfig {
	cfg := &apiConfig{
		workers:    make([]*worker, len(urls)),
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
	for i, u := range urls {
		cfg.workers[i] = &worker{url: u}
	}
	return cfg
}

func TestGetRoundRobin(t *testing.T) {
	urls3 := []string{"http://a", "http://b", "http://c"}
	cfg3 := mkConfig(urls3)

	// Expected sequence: a, b, c, a, b, c
	want3 := []string{"http://a", "http://b", "http://c", "http://a", "http://b", "http://c"}
	for i, want := range want3 {
		wk, err := cfg3.getRoundRobin()
		if err != nil {
			t.Fatalf("3-worker case: unexpected error at index %d: %v", i, err)
		}
		if wk.url != want {
			t.Fatalf("3-worker case: at index %d: got %q, want %q", i, wk.url, want)
		}
	}

	urls4 := []string{"http://a", "http://b", "http://c", "http://d"}
	cfg4 := mkConfig(urls4)
	want4 := []string{"http://a", "http://b", "http://c", "http://d",
		"http://a", "http://b", "http://c", "http://d"}
	for i, want := range want4 {
		wk, err := cfg4.getRoundRobin()
		if err != nil {
			t.Fatalf("4-worker case: unexpected error at index %d: %v", i, err)
		}
		if wk.url != want {
			t.Fatalf("4-worker case: at index %d: got %q, want %q", i, wk.url, want)
		}
	}
}

// mkConfigWithCounts builds an apiConfig with workers pre-seeded with connection counts.
func mkConfigWithCounts(counts []int32) *apiConfig {
	cfg := &apiConfig{
		workers:    make([]*worker, len(counts)),
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
	for i, v := range counts {
		w := &worker{url: fmt.Sprintf("http://localhost:800%d/infer", i)}
		w.connCount.Store(v)
		cfg.workers[i] = w
	}
	return cfg
}

func TestGetLeastConnection(t *testing.T) {
	tests := []struct {
		name   string
		counts []int32
		want   int // index of expected winning worker
	}{
		{name: "single worker", counts: []int32{3}, want: 0},
		{name: "strictly decreasing", counts: []int32{5, 4, 3, 2, 1}, want: 4},
		{name: "strictly increasing", counts: []int32{1, 2, 3, 4}, want: 0},
		{name: "middle is minimum", counts: []int32{5, 2, 7, 3}, want: 1},
		{name: "tie picks first minimum", counts: []int32{5, 2, 2, 3}, want: 1},
		{name: "all equal", counts: []int32{2, 2, 2}, want: 0},
		{name: "last is unique minimum", counts: []int32{6, 6, 6, 1}, want: 3},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := mkConfigWithCounts(tc.counts)
			wantURL := cfg.workers[tc.want].url

			got, err := cfg.getLeastConnection()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.url != wantURL {
				t.Fatalf("getLeastConnection() = %q, want %q (counts=%v)", got.url, wantURL, tc.counts)
			}
		})
	}
}

func TestMakeInferenceWorkerEviction(t *testing.T) {
	// Create a worker that points to an unreachable port
	cfg := &apiConfig{
		workers:    []*worker{{url: "http://127.0.0.1:54321"}},
		httpClient: &http.Client{Timeout: 100 * time.Millisecond}, // fast timeout for test
	}

	req, err := http.NewRequest("POST", "http://localhost:8080/infer-rr", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	// This should fail because nothing is listening on port 54321,
	// and because it's a network dial error, the worker should be evicted.
	_, err = cfg.makeInference(req, cfg.workers[0].url)
	if err == nil {
		t.Fatal("expected error from makeInference to unreachable worker, got nil")
	}

	// Verify the worker was evicted from the configuration
	cfg.mu.RLock()
	defer cfg.mu.RUnlock()
	if len(cfg.workers) != 0 {
		t.Fatalf("expected worker to be evicted, but still have %d workers", len(cfg.workers))
	}
}

