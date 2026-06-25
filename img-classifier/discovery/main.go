package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if it exists
	_ = godotenv.Load()

	schedulerURL := os.Getenv("SCHEDULER_URL")
	if schedulerURL == "" {
		log.Fatal("SCHEDULER_URL environment variable is not set")
	}

	retryStr := os.Getenv("RETRY_INTERVAL")
	retryInterval := 5 * time.Second
	if retryStr != "" {
		if d, err := time.ParseDuration(retryStr); err == nil {
			retryInterval = d
		}
	}

	regURL := fmt.Sprintf("%s/reg-worker", schedulerURL)

	fmt.Printf("[*] Starting discovery client. Targeting scheduler at %s\n", regURL)

	secret := os.Getenv("REGISTRATION_SECRET")

	for {
		req, err := http.NewRequest("POST", regURL, nil)
		if err != nil {
			fmt.Printf("[-] Failed to create registration request: %v. Retrying in %v...\n", err, retryInterval)
			time.Sleep(retryInterval)
			continue
		}
		if secret != "" {
			req.Header.Set("X-Registration-Secret", secret)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Printf("[-] Registration failed: %v. Retrying in %v...\n", err, retryInterval)
		} else {
			if resp.StatusCode == http.StatusOK {
				fmt.Printf("[+] Successfully registered with scheduler. Waiting for eviction...\n")
				// Block until the server closes the connection (evicts the worker)
				_, _ = io.ReadAll(resp.Body)
				resp.Body.Close()
				fmt.Printf("[-] Connection to scheduler closed (evicted). Re-registering...\n")
				continue
			}
			fmt.Printf("[-] Scheduler returned status: %s. Retrying in %v...\n", resp.Status, retryInterval)
			resp.Body.Close()
		}
		time.Sleep(retryInterval)
	}
}

