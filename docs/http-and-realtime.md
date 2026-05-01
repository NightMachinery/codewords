# HTTP and Realtime Backend

Milestone 4 wires the persistence, identity, wordpack, and game-engine packages into the Go server.

## Runtime wiring

`cmd/server` now opens the configured SQLite database (`CODEWORDS_DATABASE_PATH`, default `./data/codewords.sqlite`), creates the identity service, loads bundled wordpacks from `assets/wordpacks`, and serves API/WebSocket routes on `CODEWORDS_ADDR`.

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

Picture catalog endpoints currently report that no local picture catalog is available; the picture-card implementation remains part of the later wordpacks/pictures milestone.

## WebSocket API

Room sockets connect at `/ws/rooms/{roomId}` with `authToken` or `migrateId` query parameters. After authentication the server sends a viewer-specific `snapshot` immediately. Supported socket messages are:

- `ping` -> `pong`
- `setTeam` / `assignTeam`
- `toggleSpymaster`
- `toggleRepresentative`
- `startGame`
- `guessCard`
- `passTurn`
- `submitClue`
- `sendChat`

Accepted game commands are applied through `internal/game`, persisted as ordered events plus latest authoritative snapshot when a match is active, and broadcast as sanitized viewer-specific snapshots to connected clients. `startGame` can be sent over the socket or via `POST /api/rooms/{roomId}/start`. `sendChat` stores the message in SQLite and broadcasts a `chatMessage` event; the fuller chat UI remains part of the later frontend/chat milestone.

## Restart restoration

When a room runtime is first needed, the server loads the room from SQLite. Active rooms restore the latest saved authoritative snapshot. Lobby rooms are reconstructed from room metadata, settings JSON, and persisted room players.
