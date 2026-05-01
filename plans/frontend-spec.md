# Frontend Spec

## Framework

- Svelte 5 with TypeScript.
- Vite 8 build.
- Tailwind CSS built locally through the frontend toolchain.
- English UI copy only.
- No external fonts, icons, CSS frameworks from CDN, analytics, or remote assets.

## Routes

- `/` — landing/create or join room.
- `/room/:roomId` — lobby if pre-game; redirects/renders match if started.
- `/room/:roomId?migrate=...` — room-scoped migrated identity view; keep query param across refresh.
- `/match/:matchId` may exist as an alias if useful, but room URL is sufficient for v1.

## Tailwind styling rules

- Use mobile-first Tailwind utilities as the default styling approach. Add `sm:`, `md:`, and `lg:` variants only where the layout needs to change.
- Prefer semantic Svelte components with composed Tailwind class strings over global CSS files; extract repeated button, card, chip, panel, and form-control patterns into components.
- Use Tailwind design tokens for spacing, color, radius, shadows, focus rings, and typography. Use arbitrary values only when a card aspect ratio or board sizing cannot be expressed with tokens.
- Include visible focus states, disabled states, and sufficient color contrast for all interactive controls.
- Keep generated CSS local to the build output; no CDN Tailwind script or external stylesheet is allowed.

## UI screens

- Display-name prompt only when server has no saved name for the current effective identity.
- Lobby: room link copy, migrate-device copy, team columns, role badges, host settings, start button, chat.
- Game board: responsive 5x5 card grid, current team, remaining counts, pass button, role/view controls, clue editor/log, last selected card highlight, chat, game-over summary.
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
- Preserve separate card layout preferences for word and picture boards if a cards-per-row control is exposed.
- Confirmation prompts default to on for mobile-sized/touch contexts and off for desktop contexts unless the user has saved a preference.
- Optional sound preferences, picture-number badges, and spymaster picture highlights are local-only preferences.
- Store identity/display name server-side through the auth token flow.
- Never overwrite global LocalStorage auth token while using a room migrate URL.

## Card content mode UI

Pre-game host settings must expose:

- Words only.
- Images only.
- Mixed images and words.

For mixed/custom mode, show an image-card count control from 0 to 25 and explain that the rest of the 25 cards will be words. Words only sets the count to 0; Images only sets it to 25. Disable or clearly error when the local image catalog or selected wordpack cannot satisfy the requested count. The board renderer must support word cards and image cards in the same 5x5 grid.

## Clue UI

- Show a polished clue log to all players and spectators near the board, with round order, team color, clue text, clue number, status, and subtle current-round emphasis.
- Current-team spymasters get an inline clue composer while the round is open: clue text input, number control, save/update button, and validation messages.
- The clue number control supports blank in normal mode, `1..9`, and `∞` only when the room setting allows infinity clues.
- In enforced clue-limit mode, explain that clue submission is required and that the team may reveal at most the submitted numeric count.
- Avoid modal-first interactions. Use compact product UI styling, clear focus states, readable dense rows, and restrained team-color accents.
