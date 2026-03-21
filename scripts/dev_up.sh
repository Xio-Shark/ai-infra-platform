#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
mkdir -p "$ROOT_DIR/.tmp/go-cache" "$ROOT_DIR/.tmp/go-mod-cache"
cd "$ROOT_DIR"
GOCACHE="$ROOT_DIR/.tmp/go-cache" \
GOMODCACHE="$ROOT_DIR/.tmp/go-mod-cache" \
go run ./cmd/api-server
