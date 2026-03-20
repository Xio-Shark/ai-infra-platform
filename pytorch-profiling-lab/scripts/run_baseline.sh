#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
export PYTHONPATH=.
exec python -m train.train --config configs/baseline.yaml "$@"
