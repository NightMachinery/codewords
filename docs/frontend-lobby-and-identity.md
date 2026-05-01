# Frontend Lobby and Identity

Milestone 5 replaces the scaffold screen with a Svelte 5, Tailwind-styled lobby UI.

## Routes

- `/` bootstraps the browser identity, prompts for a display name when needed, creates rooms, and accepts pasted room links or ids.
- `/rooms/{roomId}` is the canonical lobby URL.
- `/room/{roomId}` remains accepted as a frontend alias.
- `/rooms/{roomId}?migrateId=...` uses the room-scoped migrate identity for that room only and does not overwrite the browser's global auth token.

The Vite dev server proxies `/api` and `/ws` to the default Go backend at `127.0.0.1:7878`.

## Identity

The browser stores a generated auth token in LocalStorage under `codewords.authToken`. The frontend calls `/api/identity/bootstrap` on load and prompts for a display name only when the server has none for that identity. Display names are saved server-side.

Room migrate links call `/api/rooms/{roomId}/migrate-bootstrap` and connect with `migrateId` in the WebSocket query string. The global LocalStorage auth token is preserved for other rooms.

## Lobby

The lobby opens a room WebSocket after the viewer has an identity. Snapshots drive team columns, role badges, settings, host permissions, and start readiness. Hosts can update wordpack, black-card count, enforced clue mode, infinity clues, roles, and team assignments. Non-host players can assign their own team.

Clipboard actions first use `navigator.clipboard`, then fall back to a temporary textarea plus `document.execCommand('copy')`, and finally show the raw link for manual copy.

Chat and gameplay board UI remain deferred to later milestones.
