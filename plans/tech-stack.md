# Tech Stack

## Final stack

- Backend language: Go, using the installed server version (`go1.22.2` observed on 2026-05-01).
- Backend framework: standard `net/http` with small mature libraries only where needed.
- Realtime: WebSockets from Go using a mature lightweight library such as `nhooyr.io/websocket` or `gorilla/websocket`.
- Storage: SQLite with WAL mode, busy timeout, and migrations.
- Frontend: Svelte 5 + TypeScript.
- Build tooling: Vite 8.
- Package manager: pnpm only.
- Static serving: Caddy serves built frontend files directly in production.
- Process management: tmux sessions created by `self_host.zsh`.
- Tests: Go tests, Vitest for frontend units, Playwright for browser flows.
- Formatting/linting: Go tooling plus Biome or the simplest mature TypeScript checker/formatter setup.

## Efficiency rationale

- Go gives low memory use, quick builds, simple deployment, good concurrency, and mature WebSocket/SQLite support.
- SQLite avoids Postgres/Redis operational overhead for this single-node self-hosted app.
- Svelte 5 + Vite 8 minimizes client bundle and development/build overhead.
- Caddy avoids running a separate static file server in production.

## Rejected for v1

- Docker: explicitly not allowed.
- Firebase or hosted realtime services: not intranet-safe.
- Captcha: unavailable in intranet and not needed for private self-hosting.
- External fonts/CDNs/assets: not offline-safe.
- Rust backend: excellent runtime efficiency, but slower compile cycles and more complexity than needed.
- boardgame.io: unnecessary for this small ruleset and not ideal for a fresh 2026 implementation.
