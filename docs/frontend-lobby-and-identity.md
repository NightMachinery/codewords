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

The lobby opens a room WebSocket after the viewer has an identity. Snapshots drive team columns, role badges, settings, host permissions, and start readiness. Moderators can update wordpacks, card content mode (words only, images only, or mixed image count), mixed image ordering, black-card count, enforced clue mode, infinity clues, observer chat, custom team names/colors, roles, and team assignments. Lobby wordpack controls use a dropdown-plus-add flow: the first selected pack remains the legacy `wordpackId`, additional selected packs are saved in `wordpackIds`, and the board draws from the duplicate-free union of all selected packs. Non-host players can assign their own team in the lobby, move themselves to observer mode during a match, and rejoin their remembered team from observer mode.

The moderator settings panel is tabbed by game mode. `Polarity` shows the original two-team setup. `Unity` switches the room into one-team co-op setup, shows the Unity team color/name, assassin count, turn limit, infinite-mode checkbox, and strict per-board budget checkbox. The finite Unity turn default is six turns per board. Unity turn-budget controls stack until the settings panel is wide enough for side-by-side options, so the checkboxes and explanatory labels do not overlap on tablet or narrow desktop widths. Switching to Unity preserves the Polarity team/spy/representative metadata server-side so switching back can restore it.

Clipboard actions first use `navigator.clipboard`, then fall back to a temporary textarea plus `document.execCommand('copy')`, and finally show the raw link for manual copy. Successful copy feedback clears itself after a short delay.

Named browser identities auto-join playable teams only while a room is still in `lobby` status. After a match is active, a previously unknown authenticated browser must choose a display name and is added to the room roster as an observer.

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

Observers are room roster members on the neutral `observers` team. They receive the same safe snapshots as non-spymaster players and cannot submit clues, reveal cards, or pass.

Unity active matches show the current active board, previous board when available, and own-board view in the bottom sticky panel. The active board is the public guessing board. The previous-board tab appears during and after handoff so players can review the board that just rotated away; the first 10 seconds of that handoff freeze clue, reveal, pass, and spy-switch actions. The own-board view lets a Unity player inspect their personal hidden board while another board is active; only that board owner receives hidden color data. For the current active board owner, the Unity tabs are previous board, current spy view, and current normal view. Own-board and previous-board viewing are read-only for guessing, and the UI asks eligible guessers to switch back to the active board before revealing. The board header shows whose board is displayed, remaining Unity/civilian/assassin counts, and remaining shared or per-board turns with card-type and turn-budget icons.

The Unity player panel uses a compact progress dashboard with active-player Unity progress, budget mode, turn pool state, active board owner, per-board Unity remaining count, and observer status during active matches. Lobby mode hides active-match progress rows. Unity team controls are mode-aware: regular players do not see team-switch buttons, moderators can move players only between Unity and Observer, and Polarity rooms never expose Unity as a team button. Player cards show all applicable role badges. The bottom sticky player strip shows only important Unity actors; the current Unity board owner is shown as the effective spy there, and if they are also a representative only the spy icon is shown in that strip. Active Unity players who are implicit guessers because no persistent rep is available show a temporary representative marker; when the only persistent rep is the current spy, the server selects one other active Unity player for that marker and guess/pass permission. Moderator badges remain visible for hosts and promoted room mods.

## Gameplay permissions and local preferences

Frontend helper logic mirrors the backend active-guesser rules:

- representatives guess/pass when a team has at least one representative;
- otherwise non-spymasters guess/pass;
- spymasters never guess/pass or reveal cards;
- observers and off-turn players are read-only.

Unity helper logic mirrors the co-op representative rules: zero reps means all non-owner Unity players can guess; one rep means only that rep can guess unless the rep owns the active board, in which case the server-selected temporary representative guesses; two or more reps means non-owner reps can guess. If no eligible guessers exist, the UI reports that Unity is waiting instead of allowing pass/guess controls.

Starting a match requires each playable team to have at least one spymaster and at least one non-spymaster guesser. Observer-team members are excluded from start requirements and cannot be made spymaster or representative.

Clue numbers support blank/any, `1..9`, and `∞` only when room settings allow infinity clues. When enforced clue mode is enabled, the UI requires a non-blank clue number before submitting and explains that guesses must wait for a numbered clue.

Local-only confirmation preferences are stored in LocalStorage under `codewords.gameplayPreferences`:

- `confirmGuesses` defaults to `true`;
- `confirmPasses` defaults to `false`.

Reveal, pass, and restart confirmations use a shared styled in-app popup instead of the browser-native confirmation dialog. Each action has tailored copy and visual emphasis; picture-card reveal confirmations include the selected card image so the guesser can verify the exact target before confirming. Restart remains a mandatory destructive confirmation while reveal/pass still respect the local preference toggles.

## Chat and picture cards

Milestone 7 adds room chat to the lobby and gameplay sidebars. Seated players can send messages; observers can send messages only when observer chat is enabled. The room load response includes recent chat history, and live WebSocket `chatMessage` events append new messages. The chat sidebar preserves its scroll position across collapse/reopen, scrolls to the newest accepted message after the local viewer sends, and follows incoming messages only when the viewer was already near the bottom.

Picture mode uses the local server catalog only. Hosts can choose words-only (`imageCardCount=0`), images-only (`imageCardCount=totalCards`), or mixed boards (`1..totalCards-1` image cards). Image cards render with `/api/pictures/{imageId}` URLs; clients never receive local filesystem paths.

Lobby moderators can set the total board size from 9 to 100 cards. The classic default is 25. Automatic hidden-color counts set neutral cards to `round(totalCards / 3)`, adjust neutral when needed so team cards minus the configured starting-team handicap can split evenly, then apply that handicap to whichever team randomly starts. Manual mode lets moderators choose blue/red base counts, neutral cards, and the same starting-team handicap; those four numbers must sum to the total. Assassins are configured as a subset of neutral cards, so the visible civilian count is neutral minus assassins.


## Final local preferences and moderator controls

LocalStorage gameplay preferences include confirmations, base board columns for mobile and desktop (defaulting to 8 desktop columns), an image-card size multiplier, an optional strict aspect-ratio mode, spymaster revealed-card style defaulting to greyed, and separate sound/visual cue toggles for chat, card reveals, incoming clues, and end-game results. Incoming clue cues fire only for submitted or updated clues, not card reveals or `NA` no-clue history rows. Mixed boards place word cards in 1×1 word-cell slots while picture cards can use compact 1×1, tall 1×2, large 2×4, or poster 4×8 footprints; the default image size is tall 1×2, and the image-size dropdown is driven from the saved numeric preference so the selected footprint remains visible. When strict aspect ratios are enabled by default, word cards fill gap-adjusted row tracks so two stacked word cards plus the grid gap align with a 1×2 image card, and image cards remain portrait; otherwise word cards keep the flexible minimum-height layout while image cards remain portrait. Board cards avoid halo shadows so spacing comes from grid gaps rather than glow, and word text is absolutely centered so the numeric badge never changes the visual center. Mixed-size cards use a single exact-ratio grid: fixed container-width-based row tracks prevent tall image cards from making an entire CSS grid row tall, and image row spans are computed from the 2:3 image ratio. The active-match header omits the internal room UUID and the old “Code grid” title; spymasters use compact SVG spy controls instead of text labels. Legacy word/image cards-per-row preferences are read as base column settings when present, but new saves use the simplified layout model. The greyed spymaster style makes revealed cards transparent while retaining color hints. Room creators are moderators by default; moderators can promote/demote other players, update room settings, assign teams/roles manually, switch the current Unity spy with an icon button, force their current board layout options to every connected player in the room with pressed-state feedback and a success toast, shuffle unrevealed card roles, reset the current clue, restart an active match back to the lobby, and use the default-on balanced random assignment for new players. Buttons have global pressed and focus-visible feedback so taps/clicks are easier to confirm.

Team display names are configurable room settings. The internal protocol still uses `blue` and `red`, but the default names shown in the UI are `Libertarians` and `Monarchists`. Custom colors and team names flow through lobby panels, player controls, turn indicators, clue rows, winner summaries, and card counts. Invalid custom color hex values fall back to the default team colors. Team color controls stack the preset trigger, hex input, and reset action within each team color row to avoid cramped desktop overlap while matching the narrow-screen layout; preset swatches open in a styled popup, while an Advanced button inside that popup exposes the browser-native color picker. The lobby and settings columns use minimum-width-safe grid tracks so an open moderator settings panel cannot force horizontal page overflow on mobile.

When a playable-team member becomes an observer, the room remembers that player’s previous team and spymaster/representative role. The observer card shows a compact rejoin control that restores the remembered assignment. Browser-local creator settings are stored per creator identity and reused for newly created rooms with a fresh seed.

## End-game memories

When a live snapshot transitions into `game_over`, each viewer receives a local-only cue based on their own result: winning-team players get a celebratory cue, losing-team players get a subdued cue, and observers get a neutral winner cue. The cue only fires on the transition, not when loading a room that already ended, and it respects the browser-local end-game sound and visual cue toggles.

Unity game-over summaries say “Unification successful.” for wins and “Players were divided.” for losses, then show Unity cards found, total turns, assassins, budget mode, and the score as Unity cards found per turn. The summary also lists every player with their personal board result and average Unity cards found per board turn, ranked from higher average to lower and using `N/A` for boards that were never played. Monality active and finished matches show a leaderboard above the player cards panel with player name, cumulative score, competition rank (equal scores share the same rank), and the score delta from the most recently closed round. The last-round delta is hidden until at least one Monality round has closed, and the bottom shortcut navigation includes Leaderboard only while viewing Monality. Unity active-spymaster changes use a stronger visual/audio cue than normal clue notifications so a player notices when their own board becomes active during an active match; lobby mode switches and restarted-lobby snapshots never emit the “Your Unity board is live.” cue. Completing one player's board shows a separate “board unified” visual cue before play rotates onward.

The game-over panel includes a **Capture Memory** button. It generates a client-side PNG from an offscreen DOM copy of the in-game board, so the keepsake uses the same centered word text, card chrome, number badges, selected-card styling, custom team colors, desktop column preferences, strict aspect-ratio setting, and image-card footprint settings that the player sees in the room. Memory captures always render at a fixed desktop width, even from mobile browsers, so exports stay wide instead of becoming narrow and tall. Before DOM-to-image rendering starts, the export path waits for fonts, images, and fitted word-card labels so the captured board keeps the same typography and words stay inside their card bounds. Word fitting sizes labels with the live board measurer and allows font glyph ink to overhang its measured line box, which avoids clipping Persian ascenders and descenders while the card itself still clips true card-bound overflow. Memory-capture exports additionally apply the `fitCardWordConservativeShrinkPx` conservative reduction, while the live board keeps the unmodified fitted size. The PNG also includes the game-finished timestamp, winner, losing team, team rosters with spymaster/representative role icons, an aurora-style background, and optional deterministic roast captions loaded from `assets/roast-packs/`; moderators can disable roasts through room settings. The export path keeps PNG as the downloaded format while using DOM-to-image rendering internally, leaving room for future SVG export without maintaining a separate hand-drawn canvas board.

## May 2026 UI/profile polish

The room header now carries compact action buttons for copying the room link, creating a migrate-device link for the current viewer, and moderator-only active-match restart. Player cards add moderator-only icon buttons for copying a migrate-device link for that card's player. General UI actions use direct per-icon Lucide imports like `lucide-svelte/icons/copy` to keep production build work smaller; bespoke game glyphs are stored as SVG files in `assets/SVG/` and consumed by the Svelte UI instead of being hand-inlined in components.

Local board layout defaults now use strict ratios, eight desktop columns, and large 2×4 image cards. The optional default-on “Board must fit height” mode applies on desktop/tablet only: it subtracts the measured bottom sticky control panel from viewport height, then narrows and centers the board so the card area fits in the remaining vertical space. Mobile keeps the board full-width.

Starting-lobby moderator settings support JSON5 setting profiles. Bundled defaults live in `assets/profiles/`, browser-saved profiles live in LocalStorage, and profile loading applies only known room setting fields so partial profiles are safe and extra fields are ignored. Bundled profiles omit wordpack, team-name, and team-color fields so applying them preserves the user's current table identity choices. The bundled Vanilla profile is a 24-word manual-count board (`8` blue base, `8` red base, `7` neutral/civilian-plus-assassin, `1` starting-team handicap, and `1` assassin) so it differs from the 25-card classic defaults. Mid-game moderator settings hide lobby-only board-generation controls and keep only live/cosmetic controls plus round tools.

The active board uses a continuous segmented remaining-count bar with SVG card-type icons. Unity's counter segments use tighter outer padding and wider interior boundary padding so the in-between segments breathe evenly without oversized end caps; the turns segment gains a stronger pending style while the displayed current board's spy still has an unspent finite turn. Image cards connect their color border directly to the image; the border thickness is controlled by `imageCardColorBorderWidthPx` in `web/src/lib/constants.ts`. Last-selected image cards show the selected green border first and the card-color frame immediately inside it, and the greyed spymaster reveal style applies opacity 30 only to revealed cards instead of dimming every disabled card. Local settings can hide card number badges.

WebSocket room actions report failed sends instead of silently dropping commands when the socket is not open. Local validation failures, socket-not-ready sends, and server command rejections append local-only **System** messages to Chat with compact debug context: action, connection state, phase, role/team, active board, board view, and timestamp. These diagnostic messages are not persisted or broadcast.

The active and game-over bottom control panel stays compact: passive turn/read-only and guesser prompts live above the board, current-team player names and role icons use the team-colored segment without repeating the team name, and the player strip moves to its own integrated color row only when the controls need the width. The strip sorts spymasters before representatives before other players, clips instead of wrapping, and shows a trailing ellipsis when clipped. Narrow controls wrap vertically instead of overlapping, Submit and Pass can collapse to icon-only buttons, and the moderator Unity spy-switch action moves out of the sticky panel into a full-width button below the board when board space is narrow. The collapse button is flush with the panel corner, and player/navigation/chat access remains available after game over while unusable action controls are hidden. Non-moderators do not see the mod settings panel or mod-only player-management controls, but still see their own usable observer/rejoin controls during an active match. Player cards show display names and role badges without exposing internal player IDs.

The production favicon is generated from `favicon/0.png` into `web/public/favicon.png` at a compressed 512×512 size and linked from `web/index.html`.

### Board fit and card chrome fixes

Board height fitting uses stable layout inputs instead of the current scroll position or a ResizeObserver loop on the board element. It still subtracts the bottom sticky panel height, but computes from a stable available board width so the board does not keep shrinking after its own max-width changes. Custom SVG glyphs continue to live in `assets/SVG/`, but are now rendered from trusted raw SVG asset text instead of CSS masks so failed masks cannot show as square blocks. Last-selected card chrome takes visual priority over color chrome: image cards show a single normal color frame until selected, then a green selected border followed by the card-color frame; word cards keep their color background but hide their color border when selected.
