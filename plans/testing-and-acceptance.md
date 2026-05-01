# Testing and Acceptance

## Backend tests

- Game engine unit tests for setup, role validation, guesses, pass, assassin, card-completion wins, and hidden views.
- Property tests for board generation invariants: 25 cards, configured assassin count, correct team counts, no duplicate selected cards.
- Auth tests for token hashing, display-name persistence, host-only actions, and migrate-link room scoping.
- Storage tests for migrations, WAL setup, restart restore, chat persistence, and event/snapshot consistency.
- WebSocket tests for connect, snapshot, command validation, broadcast, reconnect, and spectator sanitization.

## Frontend tests

- Vitest tests for URL building, clipboard fallback, identity selection with migrate URL, and view permission logic.
- Playwright flows:
  - create room, set display name, refresh without reprompt.
  - join second player, assign teams/roles, start game.
  - spymaster sees colors; guesser/spectator does not.
  - guess/pass/win flow works.
  - migrate-device link opens same room identity without changing global auth token.
  - HTTP clipboard fallback displays/copies usable links.

## Self-hosting tests

- `self_host.zsh setup` stops old sessions, builds, configures Caddy, starts production.
- `start`, `dev-start`, and `redeploy` stop conflicting prod/dev sessions first.
- Script fails clearly if required ports remain occupied.
- Caddy serves static files directly in production.
- Backend paths and WebSockets proxy correctly over HTTP.
- Existing proxy environment variables are preserved into setup/build commands and tmux where needed.

## Intranet/offline acceptance

- After dependencies are installed and assets are built, app runs without contacting public internet.
- No external font/CDN/Firebase/captcha/analytics network requests in browser devtools.
- All wordpacks are listed from local files.
- Picture mode uses only local configured images/cache.

## Documentation acceptance

- `plans/start.md` references every plan file.
- `docs/self-hosting.md` matches `self_host.zsh` behavior.
- Public URLs are dynamic or user supplied; no stale FreeBoardGames/current-running-server URLs are hardcoded.

## Mixed image/word tests

- Unit tests for `words`, `images`, and `mixed` board generation.
- Mixed mode invariant: exactly `imageCardCount` image cards and `25 - imageCardCount` word cards.
- Validation errors for invalid counts and insufficient local images/words.
- Browser test that a mixed board renders word and image cards together and all card types can be guessed/revealed.
- Restart/reconnect test proving persisted mixed card contents do not change when the picture catalog later changes.
