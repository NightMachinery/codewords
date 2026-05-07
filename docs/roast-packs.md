# Roast Packs

Roast packs are newline-delimited text files under `assets/roast-packs/`. The frontend currently imports `assets/roast-packs/roast_1.txt` as raw text for end-game memory captures.

## Format

- Write one roast template per line.
- Blank lines are ignored.
- Lines whose trimmed text starts with `#` are comments and are ignored.
- Roasts are selected deterministically from the room ID and losing team, so the same room result keeps the same caption.
- Moderators can disable memory roasts from room settings.

## Placeholders

Templates may include these placeholders:

| Placeholder | Replacement | Fallback |
| --- | --- | --- |
| `{WINNER_TEAM}` | Display name of the winning team | `Winning team` |
| `{LOSER_TEAM}` | Display name of the losing team | Losing team name |
| `{RANDOM_WINNING_PLAYER}` | Deterministic player from the winning team | Winning team name |
| `{RANDOM_LOSING_PLAYER}` | Deterministic player from the losing team | Losing team name |
| `{RANDOM_WINNING_SPYMASTER}` | Deterministic winning-team spymaster | Winning team name |
| `{RANDOM_LOSING_SPYMASTER}` | Deterministic losing-team spymaster | Losing team name |
| `{RANDOM_WINNING_GUESSER}` | Deterministic winning-team non-spymaster | Winning team name |
| `{RANDOM_LOSING_GUESSER}` | Deterministic losing-team non-spymaster | Losing team name |

Example:

```text
# Comments are skipped.
{WINNER_TEAM} solved the board; {LOSER_TEAM} got solved by it.
{RANDOM_WINNING_SPYMASTER} cooked a clue. {RANDOM_LOSING_SPYMASTER} microwaved confusion.
```
