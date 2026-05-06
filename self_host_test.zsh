#!/usr/bin/env zsh
set -euo pipefail

ROOT_DIR="${0:A:h}"
cd "$ROOT_DIR"

workdir="$(mktemp -d)"
trap 'rm -rf "$workdir"' EXIT

stubbin="$workdir/bin"
mkdir -p "$stubbin"
cat > "$stubbin/caddy" <<'STUB'
#!/usr/bin/env bash
exit 0
STUB
chmod +x "$stubbin/caddy"

export PATH="$stubbin:$PATH"
export CODEWORDS_CADDYFILE="$workdir/Caddyfile"
export CODEWORDS_DEV_PORT=5173
export CODEWORDS_ADDR=127.0.0.1:7878
export CODEWORDS_SELF_HOST_TEST_ONLY=1

source ./self_host.zsh

update_caddy "https://example.test" dev
if ! grep -q 'reverse_proxy 127.0.0.1:5173' "$CODEWORDS_CADDYFILE"; then
  echo "expected dev Caddy block to proxy frontend to Vite" >&2
  cat "$CODEWORDS_CADDYFILE" >&2
  exit 1
fi
if grep -q 'root \*' "$CODEWORDS_CADDYFILE"; then
  echo "dev Caddy block should not serve web/dist" >&2
  cat "$CODEWORDS_CADDYFILE" >&2
  exit 1
fi

update_caddy "https://example.test" prod
if ! grep -q 'root \* .*/web/dist' "$CODEWORDS_CADDYFILE"; then
  echo "prod Caddy block should serve web/dist" >&2
  cat "$CODEWORDS_CADDYFILE" >&2
  exit 1
fi
if grep -q 'reverse_proxy 127.0.0.1:5173' "$CODEWORDS_CADDYFILE"; then
  echo "prod Caddy block should remove Vite proxy" >&2
  cat "$CODEWORDS_CADDYFILE" >&2
  exit 1
fi

echo "self_host Caddy mode tests passed"
