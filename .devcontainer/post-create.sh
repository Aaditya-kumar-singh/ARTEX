#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

echo "[codespaces] Downloading Go modules..."
go mod download

echo "[codespaces] Installing frontend dependencies..."
npm --prefix web ci

echo "[codespaces] Installing ARTEX browser tooling..."
sudo npm install --global @playwright/mcp@latest @playwright/cli@latest playwright@latest
playwright install --with-deps chromium

echo "[codespaces] Development environment is ready."
