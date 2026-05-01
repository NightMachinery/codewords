# API and Realtime Contracts

## HTTP conventions

- JSON request/response bodies.
- Authenticated requests include the LocalStorage auth token, except migrate-room requests which include the room-scoped migrate id.
- Return stable machine-readable error codes plus English messages.
- All endpoints work on HTTP and HTTPS.

## Required HTTP endpoints

- `GET /healthz` — process health.
- `POST /api/identity/bootstrap` — accepts auth token, creates/fetches identity, returns user id surrogate and display name if saved.
- `POST /api/identity/display-name` — saves English/free-text display name for identity.
- `POST /api/rooms` — creates room, host membership, settings, and room link.
- `GET /api/rooms/{roomId}` — returns public room/match metadata and viewer-specific membership if authenticated.
- `POST /api/rooms/{roomId}/join` — joins room as current identity.
- `POST /api/rooms/{roomId}/settings` — host updates pre-game settings.
- `POST /api/rooms/{roomId}/start` — host starts match.
- `POST /api/rooms/{roomId}/migrate-link` — creates/reuses room-scoped migrate link for current user.
- `POST /api/rooms/{roomId}/migrate-bootstrap` — resolves room-scoped migrate id to room-local identity context.
- `GET /api/wordpacks` — lists bundled wordpacks.
- `GET /api/pictures/catalog` — lists local picture ids/availability when picture mode is enabled.

## WebSocket

- Path: `/ws/rooms/{roomId}`.
- Client connects with either global auth token or room-scoped migrate id.
- Server sends initial viewer-specific snapshot immediately after authentication.
- Client messages:
  - `setTeam`
  - `assignTeam`
  - `toggleSpymaster`
  - `toggleRepresentative`
  - `startGame`
  - `guessCard`
  - `passTurn`
  - `sendChat`
  - `ping`
- Server messages:
  - `snapshot`
  - `chatMessage`
  - `error`
  - `presence`
  - `pong`

## Snapshot rules

- Snapshot shape includes room metadata, players, teams, settings, match phase, current team, cards, winner, and viewer permissions.
- Hidden card color is present only in spymaster-authorized snapshots or after game over.
- Non-spymaster snapshots may include remaining counts but not unrevealed card colors.

## Reconnects

- Page refresh reconnects using LocalStorage auth token or migrate id from URL.
- Server treats duplicate connections for the same room identity as allowed; newest connection may become primary for presence.
- WebSocket commands are idempotent where reasonable and reject stale/invalid commands with error codes.
