#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
API_URL="${API_URL:-http://127.0.0.1:8080}"
COUNT="${COUNT:-5}"

for _ in $(seq 1 "$COUNT"); do
  curl -sS -X POST \
    -H 'Content-Type: application/json' \
    --data @"$ROOT_DIR/examples/inference_job.json" \
    "$API_URL/jobs" >/dev/null
done

curl -sS -X POST "$API_URL/dispatch/once?limit=$COUNT"
