#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
export PYTHONPATH=.
exec uvicorn server.app:app --host "${HOST:-0.0.0.0}" --port "${PORT:-8080}"
