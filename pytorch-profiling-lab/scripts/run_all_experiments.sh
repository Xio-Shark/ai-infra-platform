#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
export PYTHONPATH=.
for x in baseline exp_num_workers exp_pin_memory exp_amp exp_compile; do
  echo "=== $x ==="
  python -m "experiments.${x}"
done
