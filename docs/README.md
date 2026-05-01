# Codewords Documentation

- `specs/secretcodes-reverse-spec.md` records behavior mined from `/home/ubuntu/base/FreeBoardGames.org`.
- Implementation handoff documents live in `../plans/`.
- `self-hosting.md` documents the current `self_host.zsh` lifecycle script.
- `http-and-realtime.md` documents the Milestone 4 backend API/WebSocket wiring.

## Current implementation status

Milestones 1, 2, 3, and 4 are implemented:

- Go backend module with `GET /healthz`.
- Svelte 5 + Vite 8 + Tailwind frontend under `web/`.
- Local asset directories under `assets/wordpacks/` and `assets/pictures/`.
- Pure Go game engine under `internal/game` for lobby roles, deterministic word boards, clue rounds, turn flow, hidden snapshots, and win conditions.
- tmux/Caddy-oriented self-host skeleton in `self_host.zsh`.
- SQLite migration/storage package plus HMAC-hashed identity and room-scoped migrate-link services.
- JSON REST endpoints for identity, rooms, settings, match start, migrate links, wordpack listing, and picture catalog placeholders.
- Room WebSocket endpoint with authenticated initial snapshots, ping/pong, engine command handling, persistence, broadcast, and restart restoration from saved snapshots.

See `game-engine.md` for the current engine package boundary and behavior. See `storage-and-identity.md` for persistence and identity details. See `http-and-realtime.md` for the Milestone 4 API and realtime boundary.
