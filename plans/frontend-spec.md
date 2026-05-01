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

## Card content mode UI

Pre-game host settings must expose:

- Words only.
- Images only.
- Mixed images and words.

For mixed/custom mode, show an image-card count control from 0 to 25 and explain that the rest of the 25 cards will be words. Words only sets the count to 0; Images only sets it to 25. Disable or clearly error when the local image catalog or selected wordpack cannot satisfy the requested count. The board renderer must support word cards and image cards in the same 5x5 grid.
