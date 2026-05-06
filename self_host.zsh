#!/usr/bin/env zsh
set -euo pipefail

APP_NAME="codewords"
ROOT_DIR="${0:A:h}"
cd "$ROOT_DIR"
DEFAULT_URL="https://codewords.pinky.lilf.ir"
BACKEND_ADDR="${CODEWORDS_ADDR:-127.0.0.1:7878}"
BACKEND_PORT="${BACKEND_ADDR##*:}"
DEV_PORT="${CODEWORDS_DEV_PORT:-5173}"
PROD_SESSION="codewords-prod"
DEV_BACKEND_SESSION="codewords-dev-backend"
DEV_FRONTEND_SESSION="codewords-dev-frontend"
BIN_PATH="$ROOT_DIR/bin/codewords"
DATA_DIR="${CODEWORDS_DATA_DIR:-$ROOT_DIR/.data}"
DATABASE_PATH="${CODEWORDS_DATABASE_PATH:-$DATA_DIR/codewords.sqlite}"
IMAGE_DIR="${CODEWORDS_IMAGE_DIR:-$HOME/Pictures/SurrealPictures/chosen_2}"
IMAGE_CACHE_DIR="${CODEWORDS_IMAGE_CACHE_DIR:-$HOME/.cache/talespin/cards}"
CADDYFILE="${CODEWORDS_CADDYFILE:-$HOME/Caddyfile}"

export CODEWORDS_AVIF_PROCESS_P=n

usage() {
  cat <<USAGE
Usage: ./self_host.zsh <command> [url]

Commands:
  setup [url]      Install dependencies, build, update Caddy, and start production.
  redeploy [url]   Rebuild, update Caddy, and start production.
  start [url]      Start the production Go backend tmux session.
  stop             Stop Codewords prod/dev tmux sessions.
  dev-start [url]  Start Go backend and Vite dev server tmux sessions.

Default URL: $DEFAULT_URL
USAGE
}

tmuxnew () {
  tmux kill-session -t "$1" &> /dev/null || true
  tmux new -d -s "$@"
}

load_node() {
  if command -v nvm-load >/dev/null 2>&1; then
    nvm-load
    nvm use "${CODEWORDS_NODE_VERSION:-24}"
  fi
}

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Missing required command: $1" >&2
    exit 1
  fi
}

stop_sessions() {
  tmux kill-session -t "$PROD_SESSION" &> /dev/null || true
  tmux kill-session -t "$DEV_BACKEND_SESSION" &> /dev/null || true
  tmux kill-session -t "$DEV_FRONTEND_SESSION" &> /dev/null || true
}

port_in_use() {
  local port="$1"
  if command -v ss >/dev/null 2>&1; then
    ss -ltn "sport = :$port" | grep -q LISTEN
  else
    lsof -iTCP:"$port" -sTCP:LISTEN >/dev/null 2>&1
  fi
}

check_port_free() {
  local port="$1"
  if port_in_use "$port"; then
    echo "Port $port is already in use by a non-Codewords process." >&2
    exit 1
  fi
}

install_deps() {
  require_command go
  load_node
  require_command pnpm
  pnpm install --frozen-lockfile
}

build_all() {
  mkdir -p "$ROOT_DIR/bin" "$DATA_DIR"
  pnpm --dir "$ROOT_DIR/web" build
  go build -o "$BIN_PATH" ./cmd/server
}

caddy_url_scheme() {
  local url="$1"
  if [[ "$url" == http://* ]]; then
    echo "http"
  else
    echo "https"
  fi
}

caddy_url_host() {
  local url="${1%/}"
  url="${url#http://}"
  url="${url#https://}"
  echo "$url"
}

caddy_primary_url() {
  local url="$1"
  local scheme
  local host
  scheme="$(caddy_url_scheme "$url")"
  host="$(caddy_url_host "$url")"
  echo "$scheme://$host"
}

caddy_redirect_url() {
  local url="$1"
  local scheme
  local host
  scheme="$(caddy_url_scheme "$url")"
  host="$(caddy_url_host "$url")"
  if [[ "$scheme" == "https" ]]; then
    echo "http://$host"
  else
    echo "https://$host"
  fi
}

update_caddy() {
  local url="$1"
  require_command caddy
  mkdir -p "${CADDYFILE:h}"
  touch "$CADDYFILE"

  local primary_url
  local redirect_url
  primary_url="$(caddy_primary_url "$url")"
  redirect_url="$(caddy_redirect_url "$url")"

  local begin="# BEGIN CODEWORDS MANAGED BLOCK"
  local end="# END CODEWORDS MANAGED BLOCK"
  local tmp
  tmp="$(mktemp)"
  awk -v begin="$begin" -v end="$end" '
    $0 == begin { skip = 1; next }
    $0 == end { skip = 0; next }
    skip == 0 { print }
  ' "$CADDYFILE" > "$tmp"
  cat >> "$tmp" <<CADDY
$begin
$redirect_url {
  redir $primary_url{uri} permanent
}

$primary_url {
  encode zstd gzip
  root * $ROOT_DIR/web/dist

  handle /api/* {
    reverse_proxy $BACKEND_ADDR
  }
  handle /ws/* {
    reverse_proxy $BACKEND_ADDR
  }
  handle /healthz {
    reverse_proxy $BACKEND_ADDR
  }

  handle {
    try_files {path} /index.html
    file_server
  }
}
$end
CADDY
  mv "$tmp" "$CADDYFILE"
  caddy fmt --overwrite "$CADDYFILE"
  caddy reload --config "$CADDYFILE" || caddy start --config "$CADDYFILE"
}

start_prod() {
  local url="${1:-$DEFAULT_URL}"
  require_command tmux
  require_command go
  stop_sessions
  check_port_free "$BACKEND_PORT"
  mkdir -p "$DATA_DIR"
  if [[ ! -x "$BIN_PATH" ]]; then
    build_all
  fi
  tmuxnew "$PROD_SESSION" -c "$ROOT_DIR" -e "CODEWORDS_ADDR=$BACKEND_ADDR" -e "CODEWORDS_DATA_DIR=$DATA_DIR" -e "CODEWORDS_DATABASE_PATH=$DATABASE_PATH" -e "CODEWORDS_IMAGE_DIR=$IMAGE_DIR" -e "CODEWORDS_IMAGE_CACHE_DIR=$IMAGE_CACHE_DIR" -e "CODEWORDS_AVIF_PROCESS_P=${CODEWORDS_AVIF_PROCESS_P:-n}" -- "$BIN_PATH"
  echo "Codewords production started at $url with backend $BACKEND_ADDR"
}

start_dev() {
  local url="${1:-$DEFAULT_URL}"
  require_command tmux
  require_command go
  load_node
  require_command pnpm
  stop_sessions
  check_port_free "$BACKEND_PORT"
  check_port_free "$DEV_PORT"
  mkdir -p "$DATA_DIR"
  tmuxnew "$DEV_BACKEND_SESSION" -c "$ROOT_DIR" -e "CODEWORDS_ADDR=$BACKEND_ADDR" -e "CODEWORDS_DATA_DIR=$DATA_DIR" -e "CODEWORDS_DATABASE_PATH=$DATABASE_PATH" -e "CODEWORDS_IMAGE_DIR=$IMAGE_DIR" -e "CODEWORDS_IMAGE_CACHE_DIR=$IMAGE_CACHE_DIR" -e "CODEWORDS_AVIF_PROCESS_P=${CODEWORDS_AVIF_PROCESS_P:-n}" -- "go run ./cmd/server"
  tmuxnew "$DEV_FRONTEND_SESSION" -c "$ROOT_DIR" -- "pnpm --dir web exec vite --host 127.0.0.1 --port $DEV_PORT"
  echo "Codewords development started for $url: backend $BACKEND_ADDR, frontend 127.0.0.1:$DEV_PORT"
}

command="${1:-}"
url="${2:-$DEFAULT_URL}"

case "$command" in
  setup)
    stop_sessions
    install_deps
    build_all
    update_caddy "$url"
    start_prod "$url"
    ;;
  redeploy)
    stop_sessions
    install_deps
    build_all
    update_caddy "$url"
    start_prod "$url"
    ;;
  start)
    start_prod "$url"
    ;;
  stop)
    stop_sessions
    echo "Codewords sessions stopped."
    ;;
  dev-start)
    start_dev "$url"
    ;;
  -h|--help|help|"")
    usage
    ;;
  *)
    echo "Unknown command: $command" >&2
    usage >&2
    exit 1
    ;;
esac
