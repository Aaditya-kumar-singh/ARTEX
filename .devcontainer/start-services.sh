#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

echo "[codespaces] Waiting for PostgreSQL..."
for attempt in $(seq 1 60); do
  if pg_isready -h postgres -p 5432 -U artex -d artex >/dev/null 2>&1; then
    break
  fi
  if [ "$attempt" -eq 60 ]; then
    echo "[codespaces] PostgreSQL did not become ready." >&2
    exit 1
  fi
  sleep 2
done

if ! curl --fail --silent http://127.0.0.1:8787/api/health >/dev/null; then
  echo "[codespaces] Starting ARTEX backend..."
  nohup go run ./cmd/artex -addr 0.0.0.0:8787 -proxy 127.0.0.1:8788 \
    > .codespaces-backend.log 2>&1 < /dev/null &
  echo $! > .codespaces-backend.pid
fi

if ! curl --fail --silent http://127.0.0.1:5173/ >/dev/null; then
  echo "[codespaces] Starting Next.js frontend..."
  (
    cd web
    nohup npm run dev -- --hostname 0.0.0.0 --port 5173 \
      > ../.codespaces-frontend.log 2>&1 < /dev/null &
    echo $! > ../.codespaces-frontend.pid
  )
fi

echo "[codespaces] Waiting for ARTEX backend..."
for attempt in $(seq 1 90); do
  if curl --fail --silent http://127.0.0.1:8787/api/health >/dev/null; then
    break
  fi
  if [ "$attempt" -eq 90 ]; then
    echo "[codespaces] Backend failed to start; recent log follows:" >&2
    tail -n 80 .codespaces-backend.log 2>/dev/null || true
    exit 1
  fi
  sleep 2
done

echo "[codespaces] Waiting for ARTEX frontend..."
for attempt in $(seq 1 60); do
  if curl --fail --silent http://127.0.0.1:5173/ >/dev/null; then
    break
  fi
  if [ "$attempt" -eq 60 ]; then
    echo "[codespaces] Frontend failed to start; recent log follows:" >&2
    tail -n 80 .codespaces-frontend.log 2>/dev/null || true
    exit 1
  fi
  sleep 2
done

echo "[codespaces] ARTEX is ready. Open the forwarded port 5173."
