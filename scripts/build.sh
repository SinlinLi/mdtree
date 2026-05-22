#!/usr/bin/env bash
# Build the frontend and compile a single self-contained mdtree binary with
# the frontend embedded. Output: bin/mdtree
set -euo pipefail
cd "$(dirname "$0")/.."

VERSION="$(git describe --tags --always --dirty 2>/dev/null || echo dev)"

echo "==> Building frontend"
(cd web && npm install && npm run build)

echo "==> Building binary (version ${VERSION})"
mkdir -p bin
go build -ldflags "-X main.version=${VERSION}" -o bin/mdtree ./cmd/mdtree

echo "==> Done: bin/mdtree"
