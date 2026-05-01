# Frontend Lobby, Identity, and Gameplay

Milestones 5 and 6 replace the scaffold screen with a Svelte 5, Tailwind-styled lobby and active-match UI.

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

Named browser identities auto-join only while a room is still in `lobby` status. After a match is active, a previously unknown authenticated browser connects over WebSocket as a read-only spectator instead of being seated as a new player.

## Active match UI

When a room snapshot is `active` or `game_over`, the room route switches from lobby controls to gameplay:

- responsive 5x5 board with word cards;
- revealed card colors for all viewers, hidden-color tinting for spymasters, and all colors revealed after game over;
- last-selected card highlighting;
- current-team banner and remaining blue/red counts;
- clue composer for the current team's spymaster;
- clue log with round, team, status, number, and guesses;
- guess-by-card-click and pass controls for the active guesser;
- game-over winner summary.

Spectators are authenticated browser identities that are not seated in the match. They receive the same safe snapshots as non-spymaster players and cannot submit clues, reveal cards, or pass.

## Gameplay permissions and local preferences

Frontend helper logic mirrors the backend active-guesser rules:

- representatives guess/pass when a team has at least one representative;
- otherwise non-spymasters guess/pass;
- if a team has only spymasters, those spymasters may guess/pass;
- spectators and off-turn players are read-only.

Clue numbers support blank/any, `1..9`, and `∞` only when room settings allow infinity clues. When enforced clue mode is enabled, the UI requires a non-blank clue number before submitting and explains that guesses must wait for a numbered clue.

Local-only confirmation preferences are stored in LocalStorage under `codewords.gameplayPreferences`:

- `confirmGuesses` defaults to `true`;
- `confirmPasses` defaults to `false`.

Chat remains deferred to Milestone 7.
