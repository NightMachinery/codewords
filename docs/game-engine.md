# Game Engine

Milestone 2 adds a pure Go game engine under `internal/game`. The package owns lobby validation, balanced randomized team assignment, deterministic word/image/mixed board setup, command validation, clue rounds, hidden-information snapshots, turn flow, and win conditions. It does not depend on HTTP, WebSockets, SQLite, Svelte, or process-global state.

## Command flow

Callers create a lobby with `game.NewLobby(hostID, settings)` and apply typed commands through `game.Apply(&state, command, actorID)`. Accepted commands mutate only the supplied `State` and return a typed `Event`; rejected commands return stable sentinel errors such as `ErrForbidden`, `ErrCannotStart`, `ErrClueRequired`, or `ErrGuessLimitReached`.

Engine commands cover player seating, team assignment, observer rejoin, moderator promotion/demotion, moderator role toggles, settings updates, match start, clue submit/update, guesses, and passes. The reducer-style API is designed to map directly to later persistence events without adding storage in this milestone.

## Wordpacks and boards

`game.LoadWordpacks` reads bundled `.txt` files from `assets/wordpacks`. `game.ParseWordpack` trims whitespace and skips empty or `#` comment lines. Board generation supports configurable board sizes through `Settings.TotalCards` (`9..100`, default `25`) and words-only, images-only, and mixed boards through `Settings.ImageCardCount` (`0..TotalCards`). It requires enough unique words for the non-image cards and enough unique local image ids for the image cards, selects content deterministically from the match seed, randomizes the starting team, and assigns hidden colors.

Automatic color-count mode keeps the starting team one card ahead. It sets neutral cards to `round(totalCards / 3)`, increases neutral by one when needed so team cards are odd, splits the rest between teams, and treats assassins as a subset of neutral cards. Manual mode accepts any blue/red/neutral split whose sum equals `TotalCards`; assassins must be between zero and the neutral-card count.

## Clues

A round is one team turn. It starts when a team becomes current and ends on pass, wrong/civilian/assassin reveal, or win. Correct same-team guesses keep the same clue round open.

Current-team spymasters may submit or update a clue while the round is open. Clue text is required when submitting. Numeric clue values are `1..9`; blank is allowed only when clue limits are not enforced; infinity is allowed only when `AllowInfinityClue` is enabled. If a round ends without a submitted clue, the clue log records `NA`.

When `EnforceClueGuessLimit` is enabled, guessing is rejected until a clue with a nonblank number is submitted. Numeric clues cap accepted reveals to that number; infinity has no cap. Updates that would lower a numeric clue below already accepted guesses are rejected.

## Snapshots

`State.SnapshotFor` hides unrevealed card colors from non-spymasters and anonymous spectators during active play. Spymasters see all colors. Finished matches reveal the full board to every viewer. Clue log entries are visible to all viewers.

## Observer rejoin and restarts

Assigning a player from a playable team to observers clears their active spy/representative flags but records the previous team and role. `RejoinTeamCommand` restores that remembered playable assignment for the player or a moderator.

`RestartMatchCommand` returns the room to lobby state and increments the seed so the next start generates a fresh board instead of reusing the prior word/image order.
