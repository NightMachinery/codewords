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

The lobby opens a room WebSocket after the viewer has an identity. Snapshots drive team columns, role badges, settings, host permissions, and start readiness. Moderators can update wordpack, card content mode (words only, images only, or mixed image count), mixed image ordering, black-card count, enforced clue mode, infinity clues, observer chat, custom team names/colors, roles, and team assignments. Non-host players can assign their own team or move to observer mode.

Clipboard actions first use `navigator.clipboard`, then fall back to a temporary textarea plus `document.execCommand('copy')`, and finally show the raw link for manual copy. Successful copy feedback clears itself after a short delay.

Named browser identities auto-join only while a room is still in `lobby` status. After a match is active, a previously unknown authenticated browser connects over WebSocket as a read-only spectator instead of being seated as a new player.

## Active match UI

When a room snapshot is `active` or `game_over`, the room route switches from lobby controls to gameplay:

- responsive 5x5 board with word cards, image cards, or a mixed board;
- revealed card colors for all viewers, hidden-color tinting for spymasters, and all colors revealed after game over;
- last-selected card highlighting;
- current-team banner and remaining blue/red counts;
- clue composer for the current team's spymaster;
- clue log with round, team, status, number, and guesses;
- guess-by-card-click and pass controls for the active guesser;
- game-over winner summary with viewer-specific end-game sound/visual cues and a Capture Memory image download.

Spectators are authenticated browser identities that are not seated in the match, or anonymous `spectator=1` socket viewers. They receive the same safe snapshots as non-spymaster players and cannot submit clues, reveal cards, pass, or write chat.

## Gameplay permissions and local preferences

Frontend helper logic mirrors the backend active-guesser rules:

- representatives guess/pass when a team has at least one representative;
- otherwise non-spymasters guess/pass;
- spymasters never guess/pass or reveal cards;
- observers, spectators, and off-turn players are read-only.

Starting a match requires each playable team to have at least one spymaster and at least one non-spymaster guesser. Observer-team members are excluded from start requirements and cannot be made spymaster or representative.

Clue numbers support blank/any, `1..9`, and `∞` only when room settings allow infinity clues. When enforced clue mode is enabled, the UI requires a non-blank clue number before submitting and explains that guesses must wait for a numbered clue.

Local-only confirmation preferences are stored in LocalStorage under `codewords.gameplayPreferences`:

- `confirmGuesses` defaults to `true`;
- `confirmPasses` defaults to `false`.

## Chat and picture cards

Milestone 7 adds room chat to the lobby and gameplay sidebars. Seated players can send messages; spectators can read the log but see the composer disabled. The room load response includes recent chat history, and live WebSocket `chatMessage` events append new messages.

Picture mode uses the local server catalog only. Hosts can choose words-only (`imageCardCount=0`), images-only (`imageCardCount=25`), or mixed boards (`1..24` image cards). Image cards render with `/api/pictures/{imageId}` URLs; clients never receive local filesystem paths.


## Final local preferences and moderator controls

LocalStorage gameplay preferences include confirmations, separate word/image cards-per-row values for mobile and desktop, spymaster revealed-card style, and separate sound/visual cue toggles for chat, card reveals, incoming clues, and end-game results. Mixed boards use word and image row settings together so picture cards can stay larger while word cards remain compact. The greyed spymaster style makes revealed cards transparent while retaining color hints. Room creators are moderators by default; moderators can promote/demote other players, update room settings, assign teams/roles manually, shuffle unrevealed card roles, reset the current clue, restart an active match back to the lobby, and use the default-on balanced random assignment for new players.

Team display names are configurable room settings. The internal protocol still uses `blue` and `red`, but the default names shown in the UI are `Libertarians` and `Monarchists`. Custom colors and team names flow through lobby panels, player controls, turn indicators, clue rows, winner summaries, and card counts. Invalid custom color hex values fall back to the default team colors.

When a playable-team member becomes an observer, the room remembers that player’s previous team and spymaster/representative role. The observer card shows a compact rejoin control that restores the remembered assignment. Browser-local creator settings are stored per creator identity and reused for newly created rooms with a fresh seed.

## End-game memories

When a live snapshot transitions into `game_over`, each viewer receives a local-only cue based on their own result: winning-team players get a celebratory cue, losing-team players get a subdued cue, and spectators or observers get a neutral winner cue. The cue only fires on the transition, not when loading a room that already ended, and it respects the browser-local end-game sound and visual cue toggles.

The game-over panel includes a **Capture Memory** button. It generates a client-side PNG from the final snapshot with the room id, timestamp, winner, losing team, team rosters, and final board. Word cards render as labeled colored tiles. Picture cards try to draw the same-origin `/api/pictures/{imageId}` thumbnail and fall back to a labeled tile if an image cannot be loaded.
