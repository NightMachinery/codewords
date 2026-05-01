# Implementation Roadmap

## Milestone 1: Scaffold

- Create Go module, Svelte/Vite/Tailwind app, pnpm lockfile, basic build/test scripts, and repo docs.
- Add local asset/wordpack directories.
- Add `GET /healthz` and minimal Caddy/self-host skeleton.

## Milestone 2: Game engine

- Implement pure Go game engine for lobby settings, board setup, commands, hidden views, and win conditions.
- Add comprehensive backend tests before wiring UI.

## Milestone 3: Persistence and identity

- Add SQLite migrations, WAL setup, user token hashing, display-name persistence, rooms, players, matches, snapshots, events, and chat tables.
- Implement migrate-device link storage and room-scoped identity resolution.

## Milestone 4: HTTP and WebSocket backend

- Implement required REST endpoints.
- Implement room WebSocket hub with command validation and viewer-specific snapshots.
- Add reconnect and server restart restoration.

## Milestone 5: Frontend lobby and identity

- Implement English-only Tailwind-styled Svelte UI shell, identity bootstrap, display-name prompt, room create/join, lobby/team/role controls, room link copy, and migrate-device copy.

## Milestone 6: Frontend gameplay

- Implement Tailwind-styled responsive board, spymaster/non-spymaster/spectator views, pass/guess controls, last-selected card highlight, remaining counts, game-over state, local preferences, and dynamic WebSocket URL handling.

## Milestone 7: Wordpacks, pictures, and chat

- Copy all old wordpacks into the repo and expose them in UI.
- Implement chat with anonymous spectator read-only mode.
- Implement local picture mode if not already completed.

## Milestone 8: Self-hosting

- Complete `self_host.zsh` and `docs/self-hosting.md`.
- Ensure Caddy serves static files and Go serves API/WebSockets only in production.
- Verify tmux, proxy pass-through, port checks, HTTP support, and redeploy behavior.

## Milestone 9: Acceptance and cleanup

- Run backend/frontend/browser/self-hosting tests.
- Verify no external services/assets are referenced.
- Remove inherited creator names, donation links, propaganda, and stale URLs.
- Commit final working app in logical atomic commits.

## Mixed mode addition

Implement mixed image/word cards as part of the wordpacks/pictures milestone, before final acceptance testing. Treat it as a first-class card mode, not a later plugin: backend settings, board generation, snapshots, frontend settings UI, and tests must all support it.
