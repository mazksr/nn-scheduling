#!/usr/bin/env bash
set -euo pipefail

# Algorithms to test
algorithms=( "infer-rr" "infer-leastconn" )

# Optional: directory for CSV outputs (created if missing)
out_dir="./k6_csv/$(date +%Y-%m-%d_%H-%M-%S)"
mkdir -p "$out_dir"

# Default Prometheus Remote Write URL
export K6_PROMETHEUS_RW_SERVER_URL="${K6_PROMETHEUS_RW_SERVER_URL:-http://localhost:9090/api/v1/write}"

for i in "${!algorithms[@]}"; do
  alg="${algorithms[$i]}"
  csv_path="${out_dir}/k6_${alg}.csv"

  echo "==> Running: ALG=${alg}"
  echo "==> Writing CSV to ${csv_path} and streaming metrics to ${K6_PROMETHEUS_RW_SERVER_URL}"

  # Run with prometheus-rw (stable in newer k6); if it fails due to unrecognized output, fallback to experimental
  if ! k6 run --out "csv=${csv_path}" --out "prometheus-rw" --env "ALG=${alg}" main.ts; then
    echo "==> Fallback: attempting with experimental-prometheus-rw..."
    k6 run --out "csv=${csv_path}" --out "experimental-prometheus-rw" --env "ALG=${alg}" main.ts
  fi

  # Wait 60s between tests (skip after the last one)
  if [[ "$i" -lt $((${#algorithms[@]} - 1)) ]]; then
    echo "==> Waiting 60s before next test..."
    sleep 60
  fi
done

echo "All tests complete. CSV files in: ${out_dir}"


