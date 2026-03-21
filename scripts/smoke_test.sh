#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
API_URL="${API_URL:-http://127.0.0.1:18080}"
API_ADDR="${API_ADDR:-:18080}"
mkdir -p "$ROOT_DIR/.tmp/go-cache" "$ROOT_DIR/.tmp/go-mod-cache"

cd "$ROOT_DIR"
GOCACHE="$ROOT_DIR/.tmp/go-cache" \
GOMODCACHE="$ROOT_DIR/.tmp/go-mod-cache" \
API_ADDR="$API_ADDR" \
go run ./cmd/api-server >/tmp/ai-infra-platform-smoke.log 2>&1 &
SERVER_PID=$!
trap 'kill "$SERVER_PID" >/dev/null 2>&1 || true' EXIT

sleep 2

echo "=== 1. Create training job ==="
create_response="$(curl -sS -X POST \
  -H 'Content-Type: application/json' \
  --data @"$ROOT_DIR/examples/training_job.json" \
  "$API_URL/jobs")"
echo "$create_response"

job_id="$(printf '%s' "$create_response" | python3 -c 'import json,sys; print(json.load(sys.stdin)["id"])')"

echo "=== 2. Run job ==="
curl -sS -X POST "$API_URL/jobs/$job_id/run" | python3 -m json.tool

echo "=== 3. Check executions ==="
curl -sS "$API_URL/jobs/$job_id/executions" | python3 -m json.tool

echo "=== 4. Check trace ==="
curl -sS "$API_URL/jobs/$job_id/trace" | python3 -m json.tool

echo "=== 5. Create benchmark job ==="
bench_response="$(curl -sS -X POST \
  -H 'Content-Type: application/json' \
  --data @"$ROOT_DIR/examples/benchmark_job.json" \
  "$API_URL/jobs")"
echo "$bench_response"

bench_id="$(printf '%s' "$bench_response" | python3 -c 'import json,sys; print(json.load(sys.stdin)["id"])')"

echo "=== 6. Check metrics ==="
curl -sS "$API_URL/metrics"

echo "=== 7. Health check ==="
curl -sS "$API_URL/healthz"

echo ""
echo "smoke test passed: training=$job_id benchmark=$bench_id"
