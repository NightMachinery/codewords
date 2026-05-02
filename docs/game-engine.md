# Game Engine

Milestone 2 adds a pure Go game engine under `internal/game`. The package owns lobby validation, deterministic word-board setup, command validation, clue rounds, hidden-information snapshots, turn flow, and win conditions. It does not depend on HTTP, WebSockets, SQLite, Svelte, or process-global state.

## Command flow

Callers create a lobby with `game.NewLobby(hostID, settings)` and apply typed commands through `game.Apply(&state, command, actorID)`. Accepted commands mutate only the supplied `State` and return a typed `Event`; rejected commands return stable sentinel errors such as `ErrForbidden`, `ErrCannotStart`, `ErrClueRequired`, or `ErrGuessLimitReached`.

Milestone 2 commands cover player seating, team assignment, host role toggles, settings updates, match start, clue submit/update, guesses, and passes. The reducer-style API is designed to map directly to later persistence events without adding storage in this milestone.

## Wordpacks and boards

`game.LoadWordpacks` reads bundled `.txt` files from `assets/wordpacks`. `game.ParseWordpack` trims whitespace and skips empty or `#` comment lines. Board generation supports words-only, images-only, and mixed boards through `Settings.ImageCardCount` (`0..25`). It requires enough unique words for the non-image cards and enough unique local image ids for the image cards, selects content deterministically from the match seed, randomizes the starting team, assigns hidden colors, and clamps assassin cards to `0..8`.

## Clues

A round is one team turn. It starts when a team becomes current and ends on pass, wrong/civilian/assassin reveal, or win. Correct same-team guesses keep the same clue round open.

Current-team spymasters may submit or update a clue while the round is open. Clue text is required when submitting. Numeric clue values are `1..9`; blank is allowed only when clue limits are not enforced; infinity is allowed only when `AllowInfinityClue` is enabled. If a round ends without a submitted clue, the clue log records `NA`.

When `EnforceClueGuessLimit` is enabled, guessing is rejected until a clue with a nonblank number is submitted. Numeric clues cap accepted reveals to that number; infinity has no cap. Updates that would lower a numeric clue below already accepted guesses are rejected.

## Snapshots

`State.SnapshotFor` hides unrevealed card colors from non-spymasters and anonymous spectators during active play. Spymasters see all colors. Finished matches reveal the full board to every viewer. Clue log entries are visible to all viewers.
