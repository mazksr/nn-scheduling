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

