package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// getWorkerID maps a worker URL to a friendly name like "worker_1" based on
// its current index in the workers slice.
func (cfg *apiConfig) getWorkerID(url string) string {
	cfg.mu.RLock()
	defer cfg.mu.RUnlock()
	for i, w := range cfg.workers {
		if w.url == url {
			return fmt.Sprintf("worker_%d", i+1)
		}
	}
	return "worker_unknown"
}

// isNetworkOrDialError returns true if the error is a network/worker issue,

// and false if it is a client-side context cancellation/timeout.
func isNetworkOrDialError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	return true
}

// makeInference forwards the incoming request to the given worker URL using
// the shared HTTP client (connection pooling, keep-alives enabled).
func (cfg *apiConfig) makeInference(r *http.Request, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(r.Context(), r.Method, url, r.Body)
	if err != nil {
		return nil, err
	}
	req.Header = r.Header.Clone()
	req.Host = ""

	res, err := cfg.httpClient.Do(req)
	if err != nil {
		if isNetworkOrDialError(err) {
			cfg.removeWorker(url)
		}
		return nil, err
	}
	return res, nil
}

func sendResult(w http.ResponseWriter, res *http.Response) error {
	for key, vals := range res.Header {
		for _, v := range vals {
			w.Header().Add(key, v)
		}
	}

	w.WriteHeader(res.StatusCode)
	_, err := io.Copy(w, res.Body)
	return err
}

func (cfg *apiConfig) forward(w http.ResponseWriter, r *http.Request, url string) error {
	res, err := cfg.makeInference(r, url)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	return sendResult(w, res)
}

