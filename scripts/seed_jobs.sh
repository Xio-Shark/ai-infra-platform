#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
API_URL="${API_URL:-http://127.0.0.1:8080}"

for file in "$ROOT_DIR"/examples/*.json; do
  echo "POST $file"
  curl -sS -X POST \
    -H 'Content-Type: application/json' \
    --data @"$file" \
    "$API_URL/jobs"
  echo
done
