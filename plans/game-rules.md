# Game Rules

## Entities

- Room: pre-game container with players, host, settings, and chat.
- Match: started game with immutable initial board seed/settings and evolving game state.
- Team: blue or red.
- Roles: spymaster, representative, guesser, spectator.
- Card: word or picture id, hidden color, revealed flag.

## Lobby rules

- Host creates room and may start once both teams have at least one spymaster and all seated players are assigned to a team.
- Players may select/switch teams unless host has locked assignment in future settings.
- Host may assign teams, toggle spymaster, and toggle representative.
- Representatives are active guessers for a team when present; otherwise all non-spymasters on the current team are active guessers. If all team members are spymasters, the whole team can guess.
- In two-player mode, each player starts on a separate team as that team's spymaster and can also act as guesser for that team.
- A player can never be both spymaster and representative at the same time; assigning one role removes the other.
- Moving a player between teams clears that player's prior role flags before adding them to the destination team.
- Spectators do not occupy team seats and cannot move or chat if anonymous.

## Board setup

- Use 25 cards per match.
- Choose cards from selected wordpack or picture ids using a deterministic server-side RNG seed.
- Randomize starting team server-side.
- Assign hidden colors after selecting contents:
  - `blackCards` assassin cards, clamped from 0 to 8.
  - 8 blue cards.
  - 8 red cards.
  - 1 extra card for the starting team, giving the starting team 9 target cards.
  - Remaining cards are civilians.

## Turn rules

- Current team’s active guessers may reveal an unrevealed card.
- Spymasters may not guess unless they are active guessers because no non-spymaster/representative alternative exists.
- Revealing own team color keeps the turn.
- Revealing any other color, civilian, or assassin passes turn after applying consequences.
- Active guessers may pass, switching to the other team.
- Every accepted guess/pass increments an action id for client animation/sound sync.
- Every accepted guess records the last selected card index and selecting team so the UI can highlight the card and assassin winner resolution can credit the losing team.
- Commands with invalid card indexes, revealed cards, wrong teams, spectators, or inactive guessers are rejected server-side without mutating state.

## Clue rules

A round is one team turn. It starts when a team becomes current and ends on pass, wrong-color/civilian/assassin reveal, or win. Correct same-team guesses keep the round open.

- Current-team spymasters may submit a clue text and clue number for the open round.
- Current-team spymasters may update that clue until the round ends, including typo fixes.
- Clue text is required when submitting a clue.
- Clue number can be blank in normal mode, numeric `1..9`, or `∞` only when the `allowInfinityClue` setting is enabled.
- If a round ends without an explicit clue submission, the clue log records `NA`.
- All players and spectators can see the active clue and finalized clue log.
- A host setting `enforceClueGuessLimit` makes clue submission mandatory before guessing and rejects guesses beyond the submitted numeric clue number.
- In enforced mode, blank clue numbers are invalid. `∞` has no guess cap but still requires explicit submission and is available only when `allowInfinityClue` is enabled.
- In enforced mode, clue updates that lower the clue number below already accepted guesses for the round are rejected.

## Win conditions

- If assassin is revealed, the opposing team wins.
- If all blue cards are revealed, blue wins.
- If all red cards are revealed, red wins.
- At game over, the full board is revealed to all viewers.

## Hidden information

- Spymasters see all hidden colors during active play.
- Non-spymaster players and spectators see only revealed colors and remaining counts if useful.
- Anonymous spectators always receive a non-spymaster view and can never enable spymaster view.

## Card content mode rules

Each match has an `imageCardCount` setting from 0 to 25:

- `0`: all cards have `contentType = word`.
- `25`: all cards have `contentType = image`.
- `1..24`: exactly that many cards have `contentType = image`; all remaining cards have `contentType = word`.

Board generation order:

1. Validate enough unique words/images exist for the requested mode.
2. Deterministically select the requested number of image ids and word entries from the match seed.
3. Combine them into 25 card contents and deterministically shuffle their positions.
4. Assign hidden colors independently of content type using the normal color setup rules.

Words and images are gameplay-equivalent after board generation: guessing, revealing, turn switching, and win conditions depend only on hidden color, not content type.
