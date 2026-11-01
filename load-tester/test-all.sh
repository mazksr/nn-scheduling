#!/usr/bin/env bash
set -euo pipefail

# Algorithms to test
algorithms=( "infer-rr" "infer-leastconn" "infer-p2c" "infer-hybrid" )

# Optional: directory for CSV outputs (created if missing)
out_dir="./k6_csv/$(date +%Y-%m-%d_%H-%M-%S)"
mkdir -p "$out_dir"

# Generate images list for k6
IMAGE_DIR="${IMAGE_DIR:-/Users/mazkasr/Walls}"
echo "==> Finding images in ${IMAGE_DIR}..."
find "$IMAGE_DIR" -type f \( -iname "*.jpg" -o -iname "*.png" -o -iname "*.jpeg" \) > images.txt

if [ ! -s images.txt ]; then
  echo "Error: No images found in ${IMAGE_DIR}"
  exit 1
fi
echo "==> Found $(wc -l < images.txt) images."

for i in "${!algorithms[@]}"; do
  alg="${algorithms[$i]}"
  csv_path="${out_dir}/k6_${alg}.csv"

  echo "==> Running: ALG=${alg}"
  echo "==> Writing CSV to ${csv_path}"

  k6 run --out "csv=${csv_path}" --env "ALG=${alg}" main.ts

  # Wait 60s between tests (skip after the last one)
  if [[ "$i" -lt $((${#algorithms[@]} - 1)) ]]; then
    echo "==> Waiting 60s before next test..."
    sleep 60
  fi
done

echo "All tests complete. CSV files in: ${out_dir}"


