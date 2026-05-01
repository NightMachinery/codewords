# SecretCodes Reverse-Mined Specification

Source project: `/home/ubuntu/base/FreeBoardGames.org`

This document records behaviors observed in the original FreeBoardGames SecretCodes implementation so Codewords can preserve gameplay while moving to Go, Svelte 5, Vite, and Tailwind CSS. Observations are grounded in source files and tests; implementation-specific React, Material UI, boardgame.io, Socket.IO, GraphQL, and Next.js details are reference only and must not be copied directly.

## Technology and architecture observed

- The original game logic is a boardgame.io game in `web/src/games/secretcodes/game.ts`, with mutable helpers in `util.ts` and a React UI in `board.tsx`, `Lobby.tsx`, and `PlayBoard.tsx`.
- Localized UI strings and Material UI controls exist in the source project, but Codewords is English-only and should use Tailwind utilities/components instead of copying that UI stack.
- Wordpacks are plain `.txt` files under `web/src/games/secretcodes/wordpacks/` and are auto-discovered by filename in `constants.ts`.
- Picture mode is local-file based: the web server exposes a catalog and image endpoint, and the client maps catalog ids to cards.

## Module and directory evidence

| Concern | Evidence |
| --- | --- |
| State shape | `web/src/games/secretcodes/definitions.ts:1-53` defines teams, roles, cards, hidden colors, remaining counts, phase names, and game state fields. |
| Setup and player views | `web/src/games/secretcodes/game.ts:27-75` sets teams/cards/settings and hides unrevealed colors from non-spymasters. |
| Lobby/start/move routing | `web/src/games/secretcodes/game.ts:79-116` wires lobby and guess phases plus host/role moves. |
| Move semantics | `web/src/games/secretcodes/util.ts:63-114`, `123-145`, and `241-266` define active guessers, start validation, passing, guessing, host detection, and team moves. |
| Win conditions | `web/src/games/secretcodes/game.ts:121-142` resolves assassin, blue-complete, and red-complete wins. |
| Wordpacks | `web/src/games/secretcodes/constants.ts:6-82` parses text files, derives labels, and orders known packs. |
| Customization | `web/src/games/secretcodes/customization.tsx:143-170` exposes pictures mode, black-card count, predefined words, and custom words. |
| Picture catalog | `web/server/secretcodesPictures.ts:10-13`, `123-219`, `480-481` discover local images, cache normalized images, and require at least 25 images for availability. |
| Picture HTTP | `web/server/web.ts:91-119` exposes picture catalog and image-by-id routes. |
| UI preferences | `web/src/games/secretcodes/preferences.ts:3-64` stores cards-per-row, picture-number, highlight, confirmation, chat-sound, and card-choice-sound preferences in LocalStorage. |

## Observed requirements in EARS format

### Lobby and roles

- The system shall create exactly two teams, blue and red, for each room/match.
- When a two-player game is initialized, the system shall seat player `0` on blue as spymaster and player `1` on red as spymaster. Evidence: `game.ts:31-37`.
- The system shall allow players to switch their own team while in the lobby.
- The system shall allow only the room host to assign another player to a team. Evidence: `util.ts:10-16`.
- When a player moves from one team to another, the system shall remove that player from any spymaster or representative role on the old team before adding them to the new team. Evidence: `util.ts:245-266`.
- The system shall allow only the room host to toggle spymaster and representative roles. Evidence: `util.ts:19-46`.
- When a player is made spymaster, the system shall remove that player from representatives; when a player is made representative, the system shall remove that player from spymasters. Evidence: `util.ts:19-46` and `game.test.ts:392-401`.
- The system shall allow multiple spymasters on the same team. Evidence: `game.test.ts:160-173`.
- When representatives exist for the current team, the system shall treat only representatives as active guessers. Evidence: `util.ts:63-75` and `game.test.ts:311-331`.
- When no representatives exist, the system shall treat non-spymaster teammates as active guessers; if every teammate is a spymaster, all teammates may guess. Evidence: `util.ts:63-75`.
- The system shall allow match start only when every seated player is assigned to a team and each team has at least one spymaster. Evidence: `util.ts:80-84`.
- When the room is a local two-player game and start requirements are satisfied, the legacy UI auto-starts after picture validation; Codewords may omit auto-start for online rooms but should preserve explicit start validation. Evidence: `Lobby.tsx:81-84`.

### Board setup and settings

- The system shall generate a 25-card board for each match. Evidence: `game.ts:38-41`.
- The system shall clamp assassin/black card count to the inclusive range `0..8`. Evidence: `game.ts:29`, `util.ts:97`, `customization.tsx:153-159`.
- When starting a match, the system shall randomize the starting team. Evidence: `util.ts:94-95`.
- When assigning hidden colors, the system shall assign `blackCards` assassins, 8 blue cards, 8 red cards, and one extra card for the starting team; remaining cards are civilians. Evidence: `util.ts:97-104`.
- The system shall support zero assassin cards. Evidence: `game.test.ts:85-104`.
- The system shall reject start when the current identity is not the host or the game cannot start. Evidence: `util.ts:86-92`.
- Where word cards are used, the system shall parse each non-empty, non-comment line of a wordpack as a playable word/phrase. Evidence: `constants.ts:6-10`.
- Where wordpack labels are shown, the system shall derive simple labels from filenames without `.txt`; known packs should be ordered as English, English alternative, Dutch, Czech, German, Persian, Harry Potter, and Harry Potter Farsi before unknown packs. Evidence: `constants.ts:13-37`.

### Turn flow and win conditions

- While a match is active, only active guessers for the current team shall be allowed to reveal unrevealed cards or pass. Evidence: `util.ts:109-114`, `123-145`, `229-234`.
- When an active guesser reveals their own team color, the system shall keep the same team active. Evidence: `util.ts:138-144`.
- When an active guesser reveals any other color, a civilian, or an assassin, the system shall switch the turn to the other team after applying the reveal. Evidence: `util.ts:138-144`.
- When an active guesser passes, the system shall switch the turn to the other team. Evidence: `util.ts:109-119`.
- When a guess or pass is accepted, the system shall increment an action id and record the action type for client effects. Evidence: `util.ts:109-114`, `123-145`, `game.test.ts:369-378`.
- When a card is selected, the system shall remember the last selected card index and selecting team color for highlighting and assassin winner resolution. Evidence: `util.ts:132-137`.
- If any assassin card is revealed after play begins, the system shall declare the opposing team the winner. Evidence: `game.ts:121-131`.
- If all blue cards are revealed after play begins, the system shall declare blue the winner; if all red cards are revealed, the system shall declare red the winner. Evidence: `game.ts:133-140`.
- When the match is over, the system shall reveal the full board to all viewers. Evidence: `game.ts:58-61`.

### Hidden information and spectators

- While a match is active, the system shall send unrevealed card colors only to spymasters. Evidence: `game.ts:58-75`.
- While a viewer is not an authorized spymaster, the system shall include color only on revealed cards and may include aggregate remaining counts. Evidence: `game.ts:63-75`, `game.test.ts:410-433`.
- Anonymous spectators shall be treated as non-spymaster viewers and shall not receive a spymaster toggle or move controls. Evidence: `PlayBoard.test.tsx:225-244`.

### Pictures and mixed content

- Where picture cards are enabled, the system shall use only local image files and safe opaque image ids, never raw filesystem paths. Evidence: `server/web.ts:91-119`, `secretcodesPictures.ts:214-222`.
- Where a local picture source directory is configured, the system shall recursively discover JPG, JPEG, PNG, and WebP files and sniff extensionless supported images. Evidence: `secretcodesPictures.ts:10-13`, `232-288`.
- Where picture mode is available, the system shall require at least 25 unique normalized images in the catalog. Evidence: `secretcodesPictures.ts:10`, `214-219`.
- Where picture ids are selected, the legacy system ranks ids with a deterministic FNV-style hash of `seed:imageId` and uses id lexical order as a tie breaker. Evidence: `pictures.ts:34-64`.
- Codewords shall move picture and mixed board generation to the server so concrete card contents are persisted in match snapshots and survive reconnect/restart/catalog changes.
- Codewords shall generalize the legacy boolean `picturesMode` into canonical `imageCardCount` from 0 to 25: `0` words-only, `25` images-only, and `1..24` mixed.

### Frontend and Tailwind UI

- The frontend shall be implemented in Svelte 5 + TypeScript and styled with locally built Tailwind CSS utilities; no Tailwind CDN or external assets are allowed.
- The frontend shall use mobile-first Tailwind classes and responsive prefixes for lobby columns, the board grid, side panels, and settings controls.
- The board shall default to a 5x5 grid but retain local preferences for cards per row where supported. Evidence for legacy preferences: `preferences.ts:3-9`, `PlayBoard.tsx:239-263`.
- The frontend shall expose local toggles for confirmation prompts, chat sounds if implemented, card-choice sounds if implemented, picture card numbers, and spymaster picture highlighting. Evidence: `preferences.ts:39-64`, `PlayBoard.tsx:497-621`.
- The frontend shall default confirmation prompts to enabled on mobile and disabled on desktop when no preference exists. Evidence: `preferences.ts:47-49`.
- The frontend shall keep word-card and picture-card cards-per-row preferences independent. Evidence: `preferences.ts:3-4`, `PlayBoard.test.tsx:398-432`.

## Non-functional observations

- Server-side authorization is mandatory because the original client exposed role and view toggles but the move helpers still reject unauthorized host and guess actions.
- The new implementation should improve validation for card indexes and malformed command payloads; the original helper assumes valid indexes in some paths.
- Local/offline operation is compatible with the observed picture and wordpack model, provided all picture processing dependencies are local and documented.
- Tailwind must be installed and built as a package dependency, not loaded from a CDN, to preserve intranet/offline behavior.

## Inferred acceptance criteria

- Starting a default match with one assassin yields 9 cards for the starting team, 8 cards for the other team, 1 assassin, and 7 civilians.
- Starting a match with zero assassins yields 9 cards for the starting team, 8 for the other team, and 8 civilians.
- A non-host cannot assign teams or toggle roles; a host can assign unassigned players and can move assigned players across teams.
- Representative presence restricts guessing/passing to representatives only.
- Non-spymaster snapshots never contain unrevealed `color` values, including anonymous spectator snapshots.
- Mixed mode persists concrete word/image card contents and still uses identical reveal/pass/win rules for both content types.
- Tailwind-generated CSS is bundled into the static frontend build with no browser requests to external CSS, font, or icon hosts.

## Uncertainties and questions

- The original game has a clue-given helper but no explicit clue text in authoritative state; Codewords should decide whether clue text is chat-only or a first-class event before implementation.
- The original UI allowed host role toggles during active play; Codewords plans should preserve this unless the product owner chooses lobby-only role changes.
- The original picture cache normalizes to AVIF through a helper script. Codewords may implement a simpler local image pipeline if it preserves safe ids, local-only behavior, and documented cache paths.
- The original online room/auth flow is outside the SecretCodes game directory; Codewords replaces it with local token identities and room-scoped migrate links.

## Recommendations

1. Treat this document as the behavioral baseline for game-engine tests before any Svelte UI work.
2. Implement the Go game engine as pure functions first, including snapshot sanitization and board-generation invariants.
3. Scaffold Tailwind during the Svelte/Vite setup milestone and define reusable Svelte components for cards, buttons, panels, form controls, and status chips.
4. Keep picture catalog and board content selection server-owned from v1 to avoid the legacy client-side picture-selection weakness.
5. Preserve all copied wordpacks as data, but keep UI copy and styling native to Codewords.
