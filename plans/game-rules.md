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
- Spectators do not occupy team seats and cannot move or chat if anonymous.

## Board setup

- Use 25 cards per match.
- Choose cards from selected wordpack or picture ids using a deterministic server-side RNG seed.
- Assign hidden colors:
  - `blackCards` assassin cards, clamped from 0 to 8.
  - 8 blue cards.
  - 8 red cards.
  - 1 extra card for the starting team.
  - Remaining cards are civilians.
- Randomize starting team server-side.

## Turn rules

- Current team’s active guessers may reveal an unrevealed card.
- Spymasters may not guess unless they are active guessers because no non-spymaster/representative alternative exists.
- Revealing own team color keeps the turn.
- Revealing any other color, civilian, or assassin passes turn after applying consequences.
- Active guessers may pass, switching to the other team.
- Every accepted guess/pass increments an action id for client animation/sound sync.

## Win conditions

- If assassin is revealed, the opposing team wins.
- If all blue cards are revealed, blue wins.
- If all red cards are revealed, red wins.
- At game over, full board can be revealed to all viewers.

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
