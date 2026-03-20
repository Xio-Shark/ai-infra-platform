#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
make build
echo "Run in separate terminals: ./bin/api-server  ./bin/scheduler  ./bin/worker"
