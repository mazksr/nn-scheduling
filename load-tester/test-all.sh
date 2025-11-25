#!/usr/bin/env bash
set -euo pipefail

# Algorithms to test
algorithms=( "infer-rr" "infer-leastconn" )

# Optional: directory for CSV outputs (created if missing)
out_dir="./k6_csv/$(date +%Y-%m-%d_%H-%M-%S)"
mkdir -p "$out_dir"

for i in "${!algorithms[@]}"; do
  alg="${algorithms[$i]}"
  csv_path="${out_dir}/k6_${alg}.csv"

  echo "==> Running: ALG=${alg}"
  k6 run --out "csv=${csv_path}" --env "ALG=${alg}" main.ts

  # Wait 30s between tests (skip after the last one)
  if [[ "$i" -lt $((${#algorithms[@]} - 1)) ]]; then
    echo "==> Waiting 60s before next test..."
    sleep 60
  fi
done

echo "All tests complete. CSV files in: ${out_dir}"

