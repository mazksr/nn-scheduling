#!/usr/bin/env bash
set -e

# Start the discovery client in the background
# It will keep trying to register until the scheduler is up
/usr/local/bin/discovery_client &

# Start the FastAPI inference server using uv run
exec uv run main.py --port "${PORT:-8000}"
