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

Authentication uses explicit bearer/query/body auth tokens from browser storage. Migrate-link creation defaults to the requester, or accepts an optional `playerId` for moderator-only target-player links. Migrate bootstrap accepts only room-scoped migrate ids and never exposes the global auth token. Error responses contain stable `error.code` and English `error.message` fields.

Room settings are strictly validated by HTTP and WebSocket update paths. Dynamic board settings include `totalCards` (`9..100`, default `25`), `imageCardCount` (`0..totalCards`), `autoColorCounts`, `startingTeamHandicap`, optional manual `blueCards`/`redCards`/`neutralCards`, and `blackCards` as assassins within neutral cards. Invalid explicit counts are rejected with `invalid_settings` rather than silently clamped.

Picture catalog endpoints report whether local `.jpg`, `.jpeg`, `.png`, `.webp`, and sniffed extensionless source candidates exist. The backend serves `<imageId>.avif` cache files with long-lived cache headers; file paths are never exposed to clients. AVIF cache generation/checking runs on backend startup only when `CODEWORDS_AVIF_PROCESS_P` is truthy, or manually through `codewords avif-cache gen`. When AVIF processing is disabled, image ids and cache existence checks are deferred until match start and only run against the per-game shuffled selected source candidates plus replacements.

## WebSocket API

Room sockets connect at `/ws/rooms/{roomId}` with `authToken` or `migrateId` query parameters. After authentication the server sends a viewer-specific `snapshot` immediately, including viewer host context for lobby permissions. HTTP joins, observer additions, and settings changes broadcast fresh snapshots to connected clients. Supported socket messages are:

- `ping` -> `pong`
- `setTeam` / `assignTeam`
- `toggleSpymaster`
- `toggleRepresentative`
- `rejoinTeam`
- `toggleMod`
- `randomizeTeams`
- `updateSettings`
- `startGame`
- `guessCard`
- `passTurn`
- `submitClue`
- `switchUnitySpymaster`
- `closeMonalityRound`
- `shuffleRoles`
- `resetClue`
- `restartMatch`
- `sendChat`
- `forceBoardLayout`
- `forceTheme`

Accepted game commands are applied through `internal/game`, persisted as ordered events plus latest authoritative snapshot when a match is active, and broadcast as sanitized viewer-specific snapshots to connected clients. `randomizeTeams` is a lobby-only moderator command that balances all non-observer players between blue/red, clears representatives, and assigns one spymaster per team. `rejoinTeam` restores an observer to the previous playable team and spy/representative role remembered when they moved to observers. `switchUnitySpymaster` is a moderator-only active Unity command that cycles the current board owner to the next eligible unfinished Unity board. `closeMonalityRound` is a moderator-only active Monality command that resolves the current round immediately using the same connected-player rules as timer expiry. `updateSettings` is accepted over WebSocket for moderators, persists the room settings, applies them to the runtime state, and records an active-match event when applicable. `restartMatch` is a moderator command that returns the room to lobby status, increments and persists the room seed for a fresh next board, clears the persisted current match pointer, preserves current settings/player composition, and broadcasts a lobby snapshot. `startGame` can be sent over the socket or via `POST /api/rooms/{roomId}/start`. `sendChat` is accepted only from seated room members; observer-team members may chat only when `observerChatEnabled` is true. Authenticated non-members with display names are added to the observer roster when they load an active or finished room. `forceBoardLayout` and `forceTheme` are moderator-only commands that broadcast `boardLayoutForced` / `themeForced` to the room; neither is persisted server-side. `forceTheme` carries a `theme` field sanitized to one of the frontend theme ids (`dark`, `light`, `matrix`, `solarized-dark`, `solarized-light`, `spiderman`, `dracula`, `glitch`, `christmas-cozy`, `christmas-snow`, or `christmas-candy`, defaulting to `dark`), and receivers apply it for the current session only. See `frontend-theming.md` for the client behavior.

Unity board handoffs include `previousBoard` and `unityTransitionUntil` in snapshots when a previous board should remain visible. Monality snapshots include a `monality` payload with the current spymaster id, public attempt summaries, cumulative scores, spymaster counts, optional deadline, and final rankings; the spymaster receives the hidden `board`, while guessers receive only their own `ownAttempt` reveal state. During that transition window, the server rejects clue submission, card reveal, pass, and Unity spy-switch actions. Monality timers are in-memory and scoped to the existing single-process room runtime. When `monalityRoundSeconds` is positive, the server schedules a best-effort deadline close and persists/broadcasts the resulting snapshot.

Room snapshots decorate duplicate display names with stable room-scoped numeric suffixes so players with the same base name can be distinguished without exposing internal ids.

## Restart restoration

When a room runtime is first needed, the server loads the room from SQLite. Active rooms restore the latest saved authoritative snapshot. Lobby rooms are reconstructed from room metadata, settings JSON, and persisted room players.
