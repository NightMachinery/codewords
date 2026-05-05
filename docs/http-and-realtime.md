# HTTP and Realtime Backend

Milestone 4 wires the persistence, identity, wordpack, and game-engine packages into the Go server.

## Runtime wiring

`cmd/server` opens the configured SQLite database (`CODEWORDS_DATABASE_PATH`, default `./data/codewords.sqlite`), creates the identity service, loads bundled wordpacks from `assets/wordpacks`, discovers local pictures from `CODEWORDS_PICTURES_DIR` (default `./assets/pictures`), and serves API/WebSocket routes on `CODEWORDS_ADDR`.

## HTTP API

Implemented JSON endpoints:

- `POST /api/identity/bootstrap`
- `POST /api/identity/display-name`
- `POST /api/rooms`
- `GET /api/rooms/{roomId}`
- `POST /api/rooms/{roomId}/join`
- `POST /api/rooms/{roomId}/settings`
- `POST /api/rooms/{roomId}/start`
- `POST /api/rooms/{roomId}/migrate-link`
- `POST /api/rooms/{roomId}/migrate-bootstrap`
- `GET /api/wordpacks`
- `GET /api/pictures/catalog`
- `GET /api/pictures/{imageId}`

Authentication uses explicit bearer/query/body auth tokens from browser storage. Migrate bootstrap accepts only room-scoped migrate ids and never exposes the global auth token. Error responses contain stable `error.code` and English `error.message` fields.

Picture catalog endpoints report whether local `.jpg`, `.jpeg`, `.png`, `.webp`, and sniffed extensionless source candidates exist. The backend serves `<imageId>.avif` cache files with long-lived cache headers; file paths are never exposed to clients. AVIF cache generation/checking runs on backend startup only when `CODEWORDS_AVIF_PROCESS_P` is truthy, or manually through `codewords avif-cache gen`. When AVIF processing is disabled, image ids and cache existence checks are deferred until match start and only run against the per-game shuffled selected source candidates plus replacements.

## WebSocket API

Room sockets connect at `/ws/rooms/{roomId}` with `authToken`, `migrateId`, or `spectator=1` query parameters. After authentication the server sends a viewer-specific `snapshot` immediately, including viewer host context for lobby permissions. HTTP joins and settings changes broadcast fresh lobby snapshots to connected clients. Supported socket messages are:

- `ping` -> `pong`
- `setTeam` / `assignTeam`
- `toggleSpymaster`
- `toggleRepresentative`
- `toggleMod`
- `updateSettings`
- `startGame`
- `guessCard`
- `passTurn`
- `submitClue`
- `shuffleRoles`
- `resetClue`
- `restartMatch`
- `sendChat`

Accepted game commands are applied through `internal/game`, persisted as ordered events plus latest authoritative snapshot when a match is active, and broadcast as sanitized viewer-specific snapshots to connected clients. `updateSettings` is accepted over WebSocket for moderators, persists the room settings, applies them to the runtime state, and records an active-match event when applicable. `restartMatch` is a moderator command that returns the room to lobby status, clears the persisted current match pointer, preserves current settings/player composition, and broadcasts a lobby snapshot. `startGame` can be sent over the socket or via `POST /api/rooms/{roomId}/start`. `sendChat` is accepted only from seated room members; observer-team members may chat only when `observerChatEnabled` is true. Anonymous spectators and authenticated non-members receive snapshots but cannot write chat.

## Restart restoration

When a room runtime is first needed, the server loads the room from SQLite. Active rooms restore the latest saved authoritative snapshot. Lobby rooms are reconstructed from room metadata, settings JSON, and persisted room players.
