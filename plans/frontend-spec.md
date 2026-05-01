# Frontend Spec

## Framework

- Svelte 5 with TypeScript.
- Vite 8 build.
- English UI copy only.
- No external fonts, icons, CSS frameworks from CDN, analytics, or remote assets.

## Routes

- `/` — landing/create or join room.
- `/room/:roomId` — lobby if pre-game; redirects/renders match if started.
- `/room/:roomId?migrate=...` — room-scoped migrated identity view; keep query param across refresh.
- `/match/:matchId` may exist as an alias if useful, but room URL is sufficient for v1.

## UI screens

- Display-name prompt only when server has no saved name for the current effective identity.
- Lobby: room link copy, migrate-device copy, team columns, role badges, host settings, start button, chat.
- Game board: 5x5 card grid, current team, remaining counts, pass button, role/view controls, chat, game-over summary.
- Settings: card layout preferences, sounds if implemented, confirmation preference, picture/word mode before start.
- Spectator: read-only board and chat, no move controls, no spymaster toggle.

## Clipboard on HTTP

- First try `navigator.clipboard` when available.
- Fallback to selecting a temporary input/textarea and `document.execCommand('copy')`.
- Show the raw link for manual copy if both methods fail.

## Realtime client

- Compute WebSocket URL from `window.location`.
- Use `ws://` for `http:` and `wss://` for `https:`.
- Reconnect with backoff after transient disconnects.
- Refetch/sync snapshot after reconnect.

## Preferences

- Store purely local UI preferences in LocalStorage.
- Store identity/display name server-side through the auth token flow.
- Never overwrite global LocalStorage auth token while using a room migrate URL.
