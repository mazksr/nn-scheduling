# Build from the server directory:
#   docker build -t nn-schd-server -f server/Dockerfile server

# Build stage
FROM golang:1.26.4-alpine AS builder

WORKDIR /app

# Copy dependency manifests
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code and files
COPY . .

# Build a statically linked binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server-bin .

# Final runner stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy the compiled binary
COPY --from=builder /app/server-bin .

# The application listens on port 8080 by default
EXPOSE 8080

# Run the binary
CMD ["./server-bin"]
