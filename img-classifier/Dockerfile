# Build from the repository root:
#   docker build -t img-classifier-client -f img-classifier/Dockerfile .

# Stage 1: Build the discovery client
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY discovery/go.mod discovery/go.sum ./
RUN go mod download
COPY discovery/main.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -o discovery_client main.go

# Stage 2: Python runtime with uv
FROM ghcr.io/astral-sh/uv:python3.13-bookworm-slim

# Copy the discovery client from the builder
COPY --from=builder /app/discovery_client /usr/local/bin/discovery_client

# Set the working directory inside the container
WORKDIR /app

# Enable bytecode compilation
ENV UV_COMPILE_BYTECODE=1

# Copy dependency configuration files
COPY img-classifier/pyproject.toml img-classifier/uv.lock ./

# Install dependencies (using uv sync)
RUN uv sync --frozen --no-install-project --no-dev

# Copy the application source code
COPY img-classifier/main.py ./
COPY img-classifier/vit-base-patch16-224 ./vit-base-patch16-224
COPY img-classifier/entrypoint.sh /usr/local/bin/entrypoint.sh

RUN chmod +x /usr/local/bin/entrypoint.sh

# Run via the entrypoint script
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
