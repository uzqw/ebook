#!/usr/bin/env sh
set -eu

if [ "${1:-}" != "serve" ]; then
  exec "$@"
fi
shift || true

: "${POCKETBASE_HOST:=0.0.0.0}"
: "${POCKETBASE_PORT:=18093}"
: "${POCKETBASE_DATA_DIR:=/app/pb_data}"
: "${PUBLIC_DIR:=/app/dist}"
: "${TMPDIR:=${POCKETBASE_DATA_DIR}/tmp}"

PB_BIN="${PB_BIN:-/usr/local/bin/ebook-pocketbase}"
export TMPDIR PUBLIC_DIR

mkdir -p "$POCKETBASE_DATA_DIR" "$TMPDIR"

PB_FRONTEND_DIR="${PB_FRONTEND_DIR:-$POCKETBASE_DATA_DIR/deploy/frontend}"
PB_BACKEND_BIN="${PB_BACKEND_BIN:-$POCKETBASE_DATA_DIR/deploy/backend/ebook-pocketbase}"

if [ -d "$PB_FRONTEND_DIR" ] && [ -f "$PB_BACKEND_BIN" ]; then
  export PUBLIC_DIR="$PB_FRONTEND_DIR"
  PB_BIN="$PB_BACKEND_BIN"
  chmod 0755 "$PB_BIN" 2>/dev/null || true
fi

exec "$PB_BIN" serve \
  --http="${POCKETBASE_HOST}:${POCKETBASE_PORT}" \
  --dir="$POCKETBASE_DATA_DIR" \
  "$@"
