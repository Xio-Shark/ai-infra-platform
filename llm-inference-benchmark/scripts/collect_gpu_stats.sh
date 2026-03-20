#!/usr/bin/env bash
# Requires nvidia-smi when using NVIDIA GPUs.
set -euo pipefail
if command -v nvidia-smi >/dev/null 2>&1; then
  exec watch -n 2 nvidia-smi
else
  echo "nvidia-smi not found; install NVIDIA drivers or remove this script from your workflow."
  exit 1
fi
