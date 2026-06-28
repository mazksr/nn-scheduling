# Neural Network Inference Load Balancer & Scheduler

This repository contains a dynamic, discovery-based load-balancing and scheduling system designed for hosting and evaluating machine learning inference workloads. The architecture routes requests across multiple worker nodes (e.g., Vision Transformers and Large Language Models) and evaluates system performance using real-time metrics.

---

## 🏗️ System Architecture

```mermaid
sequenceDiagram
    participant Client as K6 Load Generator
    participant LB as Go Load Balancer (8080)
    participant Worker as Inference Worker (FastAPI / vLLM)
    participant Agent as Go Discovery Agent
    participant Prom as Prometheus (9090)

    %% Registration Flow
    Note over LB, Agent: 1. Registration Phase
    Agent->>LB: POST /reg-worker (Register Worker IP/Port)
    LB-->>Agent: 200 OK (Registration Confirmed)
    
    %% Health Check Flow
    Note over LB, Worker: 2. Periodic Health Checks
    LB->>Worker: GET /health (Every 15 seconds)
    Worker-->>LB: 200 OK (Active Status)

    %% Request/Inference Flow
    Note over Client, Worker: 3. Benchmark Run
    Client->>LB: POST /infer-rr or /infer-leastconn
    LB->>Worker: Forward Inference Request
    Worker-->>LB: Return Inference Result (ImageNet class / LLM token)
    LB-->>Client: Forward response + X-Worker-Id Header
    
    %% Real-time Metrics Stream
    Note over Client, Prom: 4. Telemetry Logging
    Client->>Prom: Push metrics via Prometheus Remote Write
```

### Core Components

1. **Go Load Balancer (`/server`)**: A lightweight reverse proxy serving as the entrypoint (port `8080`). It routes inference traffic based on scheduling algorithms like **Round Robin** and **Least Connections**, and dynamically evicts workers that fail during execution or regular health checks.
2. **Go Discovery Agent (`/discovery`)**: Runs alongside each model container. It registers the worker with the Go proxy and keeps it in the active pool.
3. **FastAPI ViT Worker (`/img-classifier`)**: Python worker exposing a Vision Transformer (`vit-base-patch16-224`) model for ImageNet classification.
4. **vLLM OpenAI Server (`/vllm-server`)**: Dockerized LLM server running `unsloth/Qwen3-1.7B-unsloth-bnb-4bit` configured with the Go discovery agent.
5. **K6 Load Tester (`/load-tester`)**: A script simulating concurrent virtual users generating image classification load. It records Time-to-First-Byte (`http_req_waiting`), request success rates, and individual worker load metrics.
6. **Prometheus Monitoring (`/prometheus`)**: Dockerized monitoring stack that collects load-testing telemetry in real time.

---

## 🛠️ Prerequisites

Ensure you have the following installed on your machine:
* **Go** (v1.23+)
* **Python 3.13+** (preferably with [uv](https://github.com/astral-sh/uv) installed)
* **Docker** & **Docker Compose**
* **k6** (v0.40.0+ for Prometheus output)

---

## 🚀 Running the Project

### Step 1: Set Up the Environment
At the root of the project, edit the `.env` file to customize settings:
```ini
WORKER_URL=http://localhost
NUM_WORKERS=4
WORKERS_PORT=8000
REGISTRATION_SECRET=change-me-before-deploying
```

### Step 2: Start the Go Load Balancer
You can compile and start the central proxy locally or run it inside a Docker container:

#### Option A: Local Go Binary
```bash
cd server
go run .
```
The server will start listening on port `8080` and wait for workers to register. Make sure a `.env` file exists in the directory or set the environment variables manually.

#### Option B: Containerized Go Server (Docker)
Build and run the load balancer inside a Docker container. In this setup, configuration is done via environment variables:
```bash
# Build the image using the server folder as context
docker build -t nn-schd-server:latest -f server/Dockerfile server

# Start the container with configuration environment variables
docker run -d \
  -p 8080:8080 \
  -e WORKERS_PORT=8000 \
  -e REGISTRATION_SECRET=change-me-before-deploying \
  --name nn-server \
  nn-schd-server:latest
```

### Step 3: Start the Inference Workers
You can run the workers locally on your machine or inside containerized environments:

#### Option A: Local Python Workers (Simulating Multi-Instance)
Start multiple instances of the image classifier on sequential ports (starting at `8000`) using the startup helper:
```bash
cd img-classifier
uv run run_script.sh
```
*Note: In another terminal, run `go run ./discovery/main.go` to register local workers with the scheduler.*

#### Option B: Containerized Docker Workers (Self-Registering)
Build the image classifier client container (which bundles both the FastAPI server and the background discovery agent):
```bash
# Build from repository root
docker build -t img-classifier-client -f img-classifier/Dockerfile .

# Start the container (Requires NVIDIA Container Toolkit on the host)
docker run -d --gpus all \
  -p 8000:8000 \
  -e SCHEDULER_URL=http://<scheduler-ip>:8080 \
  -e REGISTRATION_SECRET=change-me-before-deploying \
  img-classifier-client
```

*Note: The `--gpus all` flag is required for PyTorch to detect and use the host's GPU. Without it, or without the NVIDIA Container Toolkit installed on the host, inference will automatically fall back to the CPU.*

---

## 📊 Monitoring & Performance Benchmarking

### Step 1: Spin Up the Database Stack
Start the Prometheus database and Grafana visualization platform:
```bash
# Run from repository root
docker compose up -d
```
Prometheus will start on port `9090` and Grafana on port `3000` (default login: `admin` / `admin`).

### Step 2: Run the Benchmarks
Start the automated load test suite. The script runs benchmarks for both the **Round Robin** (`infer-rr`) and **Least Connections** (`infer-leastconn`) configurations:
```bash
cd load-tester
chmod +x test-all.sh
./test-all.sh
```
As it runs, k6 will export results to a local CSV file inside `load-tester/k6_csv/` and stream real-time metrics directly to the Prometheus database.

### Step 3: Analyze and Visualize Performance

#### A. Generate Static Plots (Python)
Use the visualizer script to parse the k6 output CSV and generate graphs showing RPS, error rates, and worker traffic distribution:
```bash
# Install visualization dependencies
uv pip install matplotlib pandas numpy

# Run visualizer
python csv-visualizer.py /path/to/k6_csv/your_test_run.csv
```
This produces three critical plots:
1. **Throughput (RPS)** over time.
2. **Tail Latency** (`http_req_waiting` average, p95, p99, and max) and failed requests.
3. **Traffic Distribution** showing how requests were routed to each worker (proves Round Robin's overlapping lines vs. Least Connection's staggered load lines).

#### B. Explore via Grafana
1. Navigate to Grafana at `http://localhost:3000`.
2. Add a new **Prometheus** Data Source targeting `http://prometheus:9090`.
3. Query metrics like `k6_http_req_waiting` and `k6_worker_1_requests` to inspect your scheduler's behavior in real time.
