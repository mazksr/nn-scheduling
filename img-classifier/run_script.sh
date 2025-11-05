#!/usr/bin/env bash
set -Eeuo pipefail

# Load .env (expects a line like: NUM_WORKERS=8)
if [[ -f .env ]]; then
  set -a
  source .env
  set +a
fi

: "${NUM_WORKERS:?NUM_WORKERS not set in .env}"
if ! [[ "$NUM_WORKERS" =~ ^[0-9]+$ ]] || (( NUM_WORKERS < 1 )); then
  echo "NUM_WORKERS must be a positive integer" >&2
  exit 1
fi

BASE_PORT="${BASE_PORT:-8000}"     # optional override via .env
PIDS_FILE=".uv_workers.pids"
: > "$PIDS_FILE"

for (( i=0; i<NUM_WORKERS; i++ )); do
  port=$(( BASE_PORT + i ))
  echo "Starting: uv run main.py --port ${port}"
  uv run main.py --port "${port}" &
  echo $! >> "$PIDS_FILE"
done

echo "Started ${NUM_WORKERS} workers on ports ${BASE_PORT}..$((BASE_PORT+NUM_WORKERS-1))"
echo "PIDs -> ${PIDS_FILE}"

trap 'echo; echo "Stopping workers..."; xargs -r kill < "$PIDS_FILE"; rm -f "$PIDS_FILE"; exit 0' INT TERM
wait
