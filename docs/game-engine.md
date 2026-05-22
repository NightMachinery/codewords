# Game Engine

Milestone 2 adds a pure Go game engine under `internal/game`. The package owns lobby validation, balanced randomized team assignment, deterministic word/image/mixed board setup, command validation, clue rounds, hidden-information snapshots, turn flow, and win conditions. It does not depend on HTTP, WebSockets, SQLite, Svelte, or process-global state.

## Modes

The engine supports two rule modes. `polarity` is the original two-team mode. `unity` is a cooperative mode with one playable Unity team and per-player hidden boards. Lobby mode switches are explicit room settings: switching from Polarity to Unity moves playable blue/red players onto Unity while preserving their previous team, spymaster, and representative metadata; switching back restores that metadata when present.

## Command flow

Callers create a lobby with `game.NewLobby(hostID, settings)` and apply typed commands through `game.Apply(&state, command, actorID)`. Accepted commands mutate only the supplied `State` and return a typed `Event`; rejected commands return stable sentinel errors such as `ErrForbidden`, `ErrCannotStart`, `ErrClueRequired`, or `ErrGuessLimitReached`.

Engine commands cover player seating, team assignment, observer rejoin, moderator promotion/demotion, moderator role toggles, settings updates, match start, clue submit/update, guesses, and passes. The reducer-style API is designed to map directly to later persistence events without adding storage in this milestone.

## Wordpacks and boards

`game.LoadWordpacks` reads bundled `.txt` files from `assets/wordpacks`. `game.ParseWordpack` trims whitespace and skips empty or `#` comment lines. Board generation supports configurable board sizes through `Settings.TotalCards` (`9..100`, default `25`) and words-only, images-only, and mixed boards through `Settings.ImageCardCount` (`0..TotalCards`). It requires enough unique words for the non-image cards and enough unique local image ids for the image cards, selects content deterministically from the match seed, randomizes the starting team, and assigns hidden colors.

Automatic color-count mode applies the configured starting-team handicap to whichever team randomly starts. It sets neutral cards to `round(totalCards / 3)`, adjusts neutral when needed so team cards minus that handicap can split evenly, splits the base team cards evenly, and treats assassins as a subset of neutral cards. Manual mode accepts blue/red base counts, neutral cards, and a starting-team handicap whose sum equals `TotalCards`; the handicap is applied to whichever team randomly starts. Assassins must be between zero and the neutral-card count.

Unity starts one deterministic board per active Unity player. Each board seed derives from the room seed, match id, and board owner id. Unity board colors are Unity, assassin, and civilian. The Unity target count is `round(totalCards / 2.5)`, and Unity defaults to four assassins and six finite turns per board when the lobby does not set those values.

## Clues

A round is one team turn. It starts when a team becomes current and ends on pass, wrong/civilian/assassin reveal, or win. Correct same-team guesses keep the same clue round open.

Current-team spymasters may submit or update a clue while the round is open. Clue text is required when submitting. Numeric clue values are `1..9`; blank is allowed only when clue limits are not enforced; infinity is allowed only when `AllowInfinityClue` is enabled. Guesses before a clue do not create a live placeholder clue row; if a round ends without a submitted clue, the clue log records `NA`.

When `EnforceClueGuessLimit` is enabled, guessing is rejected until a clue with a nonblank number is submitted. Numeric clues cap accepted reveals to that number; infinity has no cap. Updates that would lower a numeric clue below already accepted guesses are rejected.

## Snapshots

`State.SnapshotFor` hides unrevealed card colors from non-spymasters and observers during active play. Spymasters see all colors. Finished matches reveal the full board to every viewer. Clue log entries are visible to all viewers.

Unity snapshots expose the active board to everyone, but hidden colors are present only for revealed cards unless the viewer owns that board or the match is over. Unity players also receive an `ownBoard` snapshot with their own hidden colors for clue composition. Each Unity board carries a separate clue log and last-selected card marker, so board changes do not carry a selection highlight onto unrelated boards.

## Unity turn and budget rules

The active board owner is that board's spymaster. They may submit clues for their own active board but cannot guess or pass on it. With zero representatives, every non-observer Unity player except the active board owner can guess/pass. With one representative, that representative is the only guesser unless they are the active board owner; in that case all other Unity players can guess/pass. With two or more representatives, only representatives other than the active board owner can guess/pass.

Correct Unity guesses keep the same board active while that board still has unrevealed Unity cards. If a correct guess finds the last Unity card on that board, the engine finalizes that board's clue row and auto-rotates away because there is nothing useful left to guess there. Civilian guesses, pass, and enforced clue-limit exhaustion also end the board turn and rotate to the next unfinished active Unity board. Revealing any assassin ends the match as a global loss. When all active Unity players' boards have no unrevealed Unity cards, Unity wins.

Default finite Unity difficulty uses a shared-pool ledger: the match starts with `unityTurnLimit * activeUnityBoardCount`, each active board handoff spends one turn, and solved boards leave unused turns in the pool. Auto-rotating away from a just-solved board does not charge that solved board again. If a player becomes an observer, their unspent contribution is withdrawn from the pool but clamped at zero; past turns never create retroactive debt. If the pool is zero after roster reconciliation, Unity wins if all active boards are solved, loses if unresolved active boards can proceed, or waits if there are no eligible guessers. Rejoining restores the existing board and only restores the contribution previously withdrawn for that player.

Strict per-board mode gives each board its own `unityTurnLimit`; solved boards lose unused turns, and observer/rejoin affects only that board.

## Observer rejoin and restarts

Assigning a player from a playable team to observers clears their active spy/representative flags but records the previous team and role. `RejoinTeamCommand` restores that remembered playable assignment for the player or a moderator.

In active Unity matches, new link-openers are added as observers. If a moderator assigns a new observer to Unity, the engine creates a fresh personal board and adjusts the shared pool by the configured turn limit when shared-pool mode is active. Returning observers keep their board state.

Unity lobby joins assign new players directly to Unity instead of applying Polarity team balancing. Unity image-card boards use a deterministic match-wide image pool: images are not duplicated across boards when enough unique assets exist, and reuse starts only when the available pool is exhausted.

Unity end stats include global found/total/turn/assassin totals plus one row per personal board with owner id, Unity cards found, total Unity cards, turns used, and Unity cards found per board turn. Unplayed boards expose no per-turn average.

`RestartMatchCommand` returns the room to lobby state, clears active board state including Unity personal boards and progress, and increments the seed so the next start generates a fresh board instead of reusing the prior word/image order.
