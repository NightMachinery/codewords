# Architecture

## Repository layout

Recommended top-level structure:

- `cmd/server/` — Go server entrypoint.
- `internal/` — Go packages for config, HTTP handlers, WebSocket hub, game engine, persistence, auth, chat, assets.
- `web/` — Svelte 5 + Vite frontend.
- `assets/wordpacks/` — copied SecretCodes wordpacks.
- `assets/pictures/` or configured local picture directory support.
- `migrations/` — SQLite migrations.
- `docs/self-hosting.md` — operator docs.
- `self_host.zsh` — self-host lifecycle script.
- `plans/` — these planning docs.

## Runtime topology

Production:

- Caddy listens on the public URL.
- Caddy serves `web/dist` directly.
- Caddy reverse-proxies `/api/*`, `/ws/*`, `/healthz`, and dynamic local picture endpoints to Go.
- Go owns API, WebSocket, game state validation, persistence, and local dynamic picture access.
- SQLite database lives in a configured local data directory.

Development:

- Go server runs in a tmux dev session.
- Vite dev server runs in a tmux dev session for frontend hot reload.
- Caddy may proxy frontend to Vite and backend paths to Go, or developers may open Vite directly.

## Data flow

- Client bootstraps identity from LocalStorage token or a room-scoped migrate id in the URL.
- Client calls HTTP APIs for identity, room creation/joining, initial snapshots, and migrate-link creation.
- Client opens one WebSocket per room/match for commands and broadcasts.
- Server validates every command against persisted room/match state.
- Server broadcasts role-appropriate snapshots; clients never receive hidden card colors unless authorized.

## URL handling

- Runtime server URL is derived from the current browser location.
- No current-running-server URL is hardcoded in frontend code.
- WebSocket URL derives protocol dynamically: `http:` -> `ws:`, `https:` -> `wss:`.
