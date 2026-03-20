#!/usr/bin/env bash
set -euo pipefail
BASE="${API:-http://127.0.0.1:8080}"
code=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/healthz")
[[ "$code" == "200" ]] || exit 1
echo "ok healthz"
