#!/usr/bin/env bash
set -euo pipefail

# PiHolster — development launcher.
# Starts the Go backend on :8080 and the SvelteKit frontend on :5173.
# Both processes run in the background; Ctrl-C (SIGINT) stops both cleanly.

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKEND_LOG="$REPO_ROOT/tmp/backend.log"
FRONTEND_LOG="$REPO_ROOT/tmp/frontend.log"

mkdir -p "$REPO_ROOT/tmp"

BACKEND_PID=""
FRONTEND_PID=""

cleanup() {
    echo ""
    echo "Shutting down dev servers..."
    [ -n "$BACKEND_PID" ]  && kill "$BACKEND_PID"  2>/dev/null || true
    [ -n "$FRONTEND_PID" ] && kill "$FRONTEND_PID" 2>/dev/null || true
    wait 2>/dev/null || true
    echo "Done."
}
trap cleanup INT TERM EXIT

echo "==> Building backend..."
cd "$REPO_ROOT"
mkdir -p bin
go build -o bin/piholsterd ./apps/piholsterd/cmd/piholsterd

echo "==> Starting backend  (log: tmp/backend.log)"
./bin/piholsterd >"$BACKEND_LOG" 2>&1 &
BACKEND_PID=$!

echo "==> Starting frontend (log: tmp/frontend.log)"
pnpm --filter web dev >"$FRONTEND_LOG" 2>&1 &
FRONTEND_PID=$!

echo ""
echo "Backend  PID=$BACKEND_PID   http://localhost:8080"
echo "Frontend PID=$FRONTEND_PID  http://localhost:5173"
echo ""
echo "Press Ctrl-C to stop both servers."
echo ""

# Exit when either process dies
wait -n "$BACKEND_PID" "$FRONTEND_PID" 2>/dev/null || true
