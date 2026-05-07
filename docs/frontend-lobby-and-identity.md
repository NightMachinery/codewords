# Frontend Lobby, Identity, and Gameplay

Milestones 5 and 6 replace the scaffold screen with a Svelte 5, Tailwind-styled lobby and active-match UI.

## Routes

- `/` bootstraps the browser identity, prompts for a display name when needed, creates rooms, and accepts pasted room links or ids. The landing surface is intentionally minimal: CODEWORDS branding, the room entry controls, and a CSS-only animated aurora background with reduced-motion handling.
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

- responsive board with word cards, image cards, or a mixed board; the active board uses the full desktop row before player and clue panels, and the mobile shell avoids reserved chat padding so the board can use the available screen width;
- revealed card colors for all viewers, hidden-color tinting for spymasters, and all colors revealed after game over; word labels shrink or enlarge inside their cards, use a glyph-safe line height for Persian/Arabic scripts, and only spaces or Persian half-spaces create deliberate wrap opportunities;
- last-selected card highlighting;
- current-team banner and remaining blue/red counts;
- lobby start controls use a minimal sticky bottom panel on both desktop and mobile so the team and settings panels can stay focused on setup;
- collapsible fixed bottom controls with shortcut buttons for Board, Players, Clues, Mod Settings, Local Settings, and Chat; Players/Mod Settings scroll to their panel anchors, and Chat toggles the sidebar open or closed;
- a compact bottom-panel current-team row with a colored turn circle and a glow when the viewer can act;
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

Picture mode uses the local server catalog only. Hosts can choose words-only (`imageCardCount=0`), images-only (`imageCardCount=totalCards`), or mixed boards (`1..totalCards-1` image cards). Image cards render with `/api/pictures/{imageId}` URLs; clients never receive local filesystem paths.

Lobby moderators can set the total board size from 9 to 100 cards. The classic default is 25. Automatic hidden-color counts set neutral cards to `round(totalCards / 3)`, adjust neutral when needed so team cards minus the configured starting-team handicap can split evenly, then apply that handicap to whichever team randomly starts. Manual mode lets moderators choose blue/red base counts, neutral cards, and the same starting-team handicap; those four numbers must sum to the total. Assassins are configured as a subset of neutral cards, so the visible civilian count is neutral minus assassins.


## Final local preferences and moderator controls

LocalStorage gameplay preferences include confirmations, base board columns for mobile and desktop (defaulting to 8 desktop columns), an image-card size multiplier, an optional strict aspect-ratio mode, spymaster revealed-card style, and separate sound/visual cue toggles for chat, card reveals, incoming clues, and end-game results. Mixed boards place word cards in 1×1 word-cell slots while picture cards can use compact 1×1, tall 1×2, large 2×4, or poster 4×8 footprints; the default image size is tall 1×2, and the image-size dropdown is driven from the saved numeric preference so the selected footprint remains visible. When strict aspect ratios are enabled by default, word cards fill gap-adjusted row tracks so two stacked word cards plus the grid gap align with a 1×2 image card, and image cards remain portrait; otherwise word cards keep the flexible minimum-height layout while image cards remain portrait. Board cards avoid halo shadows so spacing comes from grid gaps rather than glow, and word text is absolutely centered so the numeric badge never changes the visual center. Mixed-size cards use a single exact-ratio grid: fixed container-width-based row tracks prevent tall image cards from making an entire CSS grid row tall, and image row spans are computed from the 2:3 image ratio. The active-match header omits the internal room UUID and the old “Code grid” title; spymasters use a compact SVG spy-view toggle instead of a text label. Legacy word/image cards-per-row preferences are read as base column settings when present, but new saves use the simplified layout model. The greyed spymaster style makes revealed cards transparent while retaining color hints. Room creators are moderators by default; moderators can promote/demote other players, update room settings, assign teams/roles manually, force their current board layout options to every connected player in the room with pressed-state feedback and a success toast, shuffle unrevealed card roles, reset the current clue, restart an active match back to the lobby, and use the default-on balanced random assignment for new players. Buttons have global pressed and focus-visible feedback so taps/clicks are easier to confirm.

Team display names are configurable room settings. The internal protocol still uses `blue` and `red`, but the default names shown in the UI are `Libertarians` and `Monarchists`. Custom colors and team names flow through lobby panels, player controls, turn indicators, clue rows, winner summaries, and card counts. Invalid custom color hex values fall back to the default team colors. Team color controls stack the preset trigger, hex input, and reset action within each team color row to avoid cramped desktop overlap while matching the narrow-screen layout; preset swatches open in a styled popup, while an Advanced button inside that popup exposes the browser-native color picker. The lobby and settings columns use minimum-width-safe grid tracks so an open moderator settings panel cannot force horizontal page overflow on mobile.

When a playable-team member becomes an observer, the room remembers that player’s previous team and spymaster/representative role. The observer card shows a compact rejoin control that restores the remembered assignment. Browser-local creator settings are stored per creator identity and reused for newly created rooms with a fresh seed.

## End-game memories

When a live snapshot transitions into `game_over`, each viewer receives a local-only cue based on their own result: winning-team players get a celebratory cue, losing-team players get a subdued cue, and spectators or observers get a neutral winner cue. The cue only fires on the transition, not when loading a room that already ended, and it respects the browser-local end-game sound and visual cue toggles.

The game-over panel includes a **Capture Memory** button. It generates a client-side PNG from the final snapshot with timestamp, winner, losing team, team rosters, an aurora background, and the board rendered like a normal player saw it: revealed cards keep color, unrevealed cards stay hidden, and last-selected styling is preserved. Optional deterministic roast captions are loaded from `assets/roast-packs/roast_1.txt`; moderators can disable them through room settings. Word cards render as labeled colored tiles. Picture cards try to draw the same-origin `/api/pictures/{imageId}` thumbnail and fall back to a labeled tile if an image cannot be loaded.

## May 2026 UI/profile polish

The room header now carries compact action buttons for copying the room link, creating a migrate-device link, and moderator-only active-match restart. General UI actions use Lucide icons; bespoke game glyphs are stored as SVG files in `assets/SVG/` and consumed by the Svelte UI instead of being hand-inlined in components.

Local board layout defaults now use strict ratios, eight desktop columns, and large 2×4 image cards. The optional default-on “Board must fit height” mode applies on desktop/tablet only: it subtracts the measured bottom sticky control panel from viewport height, then narrows and centers the board so the card area fits in the remaining vertical space. Mobile keeps the board full-width.

Starting-lobby moderator settings support JSON5 setting profiles. Bundled defaults live in `assets/profiles/`, browser-saved profiles live in LocalStorage, and profile loading applies only known room setting fields so partial profiles are safe and extra fields are ignored. Bundled profiles omit wordpack, team-name, and team-color fields so applying them preserves the user's current table identity choices. The bundled Vanilla profile is a 24-word manual-count board (`8` blue base, `8` red base, `7` neutral/civilian-plus-assassin, `1` starting-team handicap, and `1` assassin) so it differs from the 25-card classic defaults. Mid-game moderator settings hide lobby-only board-generation controls and keep only live/cosmetic controls plus round tools.

The active board uses a continuous segmented remaining-count bar with SVG card-type icons. Image cards connect their color border directly to the image, last-selected cards keep the outer selection treatment while card color becomes an immediate inner border, and the greyed spymaster reveal style is opacity-only instead of adding grey overlays. Local settings can hide card number badges.

The active and game-over bottom control panel stays compact: passive turn/read-only messages live above the board, the collapse button sits in the panel corner, and player/navigation/chat access remains available after game over while unusable action controls are hidden. Non-moderators do not see the mod settings panel or mod-only player-management controls.

The production favicon is generated from `favicon/0.png` into `web/public/favicon.png` at a compressed 512×512 size and linked from `web/index.html`.

### Board fit and card chrome fixes

Board height fitting uses stable layout inputs instead of the current scroll position or a ResizeObserver loop on the board element. It still subtracts the bottom sticky panel height, but computes from a stable available board width so the board does not keep shrinking after its own max-width changes. Custom SVG glyphs continue to live in `assets/SVG/`, but are now rendered from trusted raw SVG asset text instead of CSS masks so failed masks cannot show as square blocks. Last-selected card chrome takes visual priority over color chrome: image cards show a single normal color frame until selected, then a thick inner color frame plus the selected border; word cards keep their color background but hide their color border when selected.
