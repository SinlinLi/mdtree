#!/usr/bin/env bash
# Run the mdtree backend and the Vite dev server together for local
# development. The Vite dev server (http://localhost:5173) proxies API calls
# to the backend (http://localhost:8080), giving frontend hot-reload.
#
# Any extra arguments are forwarded to the backend, e.g.:
#   ./scripts/dev.sh --root ~/notes
set -euo pipefail
cd "$(dirname "$0")/.."

cleanup() { kill 0 2>/dev/null || true; }
trap cleanup EXIT INT TERM

if [ ! -d web/node_modules ]; then
  echo "==> Installing frontend dependencies"
  (cd web && npm install)
fi

echo "==> Backend  : http://localhost:8080"
go run ./cmd/mdtree "$@" &

echo "==> Frontend : http://localhost:5173 (open this one)"
(cd web && npm run dev) &

wait
