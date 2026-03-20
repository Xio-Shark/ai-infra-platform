#!/usr/bin/env bash
set -euo pipefail
BASE="${API:-http://127.0.0.1:8080}"
for f in examples/*.json; do
  curl -sS -X POST "$BASE/api/v1/jobs" -H 'Content-Type: application/json' -d @"$f" | head -c 200
  echo
done
