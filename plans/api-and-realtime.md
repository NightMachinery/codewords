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
- `GET /api/pictures/{imageId}` — serves a cached/normalized local picture by safe opaque id with cache headers; never accepts filesystem paths.

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
  - `submitClue`
  - `sendChat`
  - `ping`
- Server messages:
  - `snapshot`
  - `chatMessage`
  - `error`
  - `presence`
  - `pong`

## Snapshot rules

- Snapshot shape includes room metadata, players, teams, settings, match phase, current team, cards, winner, last action id/type, last selected card, remaining counts, clue log/current clue, and viewer permissions.
- Hidden card color is present only in spymaster-authorized snapshots or after game over.
- Non-spymaster snapshots may include remaining counts but not unrevealed card colors.

## Reconnects

- Page refresh reconnects using LocalStorage auth token or migrate id from URL.
- Server treats duplicate connections for the same room identity as allowed; newest connection may become primary for presence.
- WebSocket commands are idempotent where reasonable and reject stale/invalid commands with error codes.

## Card content settings API

Room settings include:

- `imageCardCount`: integer 0–25; this is the backend source of truth for words-only, images-only, and mixed boards.
- `wordpackId`: required whenever `imageCardCount < 25`.
- `enforceClueGuessLimit`: when true, current-team guessers cannot reveal cards until the current-team spymaster has submitted a clue with a nonblank number, and accepted guesses cannot exceed that number.
- `allowInfinityClue`: when true, clue submission may use `∞`; when false, infinity clues are rejected.

Settings/start validation must return clear errors for insufficient unique words, insufficient local images, unavailable picture catalog, invalid image count, invalid clue number, missing mandatory clue, or exhausted clue guess limit. Snapshots represent each card with `contentType` plus either `word` or `imageId`/image URL metadata appropriate for that viewer.
