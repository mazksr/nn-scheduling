#!/usr/bin/env bash
set -euo pipefail

# Algorithms to test
algorithms=( "infer-rr" "infer-leastconn" )

# Optional: directory for CSV outputs (created if missing)
out_dir="./k6_csv"
mkdir -p "$out_dir"

for i in "${!algorithms[@]}"; do
  alg="${algorithms[$i]}"
  ts="$(date +%Y%m%d-%H%M%S)"
  csv_path="${out_dir}/k6_${alg}_${ts}.csv"

  echo "==> Running: ALG=${alg}"
  k6 run --out "csv=${csv_path}" --env "ALG=${alg}" main.ts

  # Wait 30s between tests (skip after the last one)
  if [[ "$i" -lt $((${#algorithms[@]} - 1)) ]]; then
    echo "==> Waiting 30s before next test..."
    sleep 30
  fi
done

echo "All tests complete. CSV files in: ${out_dir}"

