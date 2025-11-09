package main

import (
	"testing"
)

func TestGetAndNext(t *testing.T) {
	numWorkers = 3
	cfg := &apiConfig{
		workers: make([]worker, numWorkers),
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


	numWorkers = 4
	cfg = &apiConfig{
		workers: make([]worker, numWorkers),
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
		workers: make([]worker, 3),
	}
	numWorkers = 3
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

func mkWorkers(counts []int32) []worker {
	ws := make([]worker, len(counts))
	for i, v := range counts {
		ws[i].connCount.Store(v)
	}
	return ws
}

func TestGetLeastConnection(t *testing.T) {
	tests := []struct {
		name   string
		counts []int32
		want   int
	}{
		{
			name:   "single worker",
			counts: []int32{3},
			want:   0,
		},
		{
			name:   "strictly decreasing",
			counts: []int32{5, 4, 3, 2, 1},
			want:   4,
		},
		{
			name:   "strictly increasing",
			counts: []int32{1, 2, 3, 4},
			want:   0,
		},
		{
			name:   "middle is minimum",
			counts: []int32{5, 2, 7, 3},
			want:   1,
		},
		{
			name:   "tie picks first minimum",
			counts: []int32{5, 2, 2, 3},
			want:   1,
		},
		{
			name:   "all equal",
			counts: []int32{2, 2, 2},
			want:   0,
		},
		{
			name:   "last is unique minimum",
			counts: []int32{6, 6, 6, 1},
			want:   3,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			numWorkers = len(tc.counts)
			cfg := &apiConfig{
				workers: mkWorkers(tc.counts),
			}
			cfg.initializeWorkers()

			got := cfg.getLeastConnection()
			if got != tc.want {
				t.Fatalf("updateLeastConnection() = %d, want %d (counts=%v)", got, tc.want, tc.counts)
			}
		})
	}
}
