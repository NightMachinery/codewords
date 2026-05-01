# Product Spec

## Goal

Build Codewords: a fast, efficient, standalone, self-hostable SecretCodes-style game for local networks and private web hosting. It should preserve current FreeBoardGames SecretCodes gameplay while removing legacy framework constraints.

## Supported gameplay

- Online rooms with shareable room links.
- Lobby with host controls, team assignment, role assignment, and start-game validation.
- Two teams: blue and red.
- Roles: spymaster, representative, regular guesser, spectator.
- Word-card mode with 25 cards.
- Picture-card mode using local image sources/cache only.
- 0–8 assassin/black cards configurable before start.
- Team turn flow, guessing, passing, wrong-guess turn switching, action cues, last-card highlighting, and win/loss detection.
- Chat for room/match participants and read-only chat for anonymous spectators.
- Reconnect and page refresh support.
- Finished-game state with full-board reveal and play-again/new-room affordance.

## UI and language

- UI text is English only.
- Do not implement runtime UI localization or translation infrastructure for v1.
- Wordpacks are data, not UI. Copy existing SecretCodes wordpacks directly, including non-English packs.
- Use Tailwind CSS for all new frontend styling. Tailwind must be installed and built locally; do not load Tailwind, fonts, icons, or component styles from a CDN.

## Intranet/offline requirements

- App must be usable without public internet after setup.
- Bundle all fonts, icons, CSS, images, and app assets locally.
- Do not use Google services, Firebase, captcha, external CDNs, remote analytics, remote image services, donation links, or externally hosted media.
- The site must work over plain HTTP.
- Clipboard features must have an HTTP-compatible fallback.
- WebSockets must dynamically use `ws://` on HTTP pages and `wss://` on HTTPS pages.

## Content cleanup

- Use this project’s own Codewords branding and repository URLs.
- Remove previous creator names from copied UI/content.
- Remove donation links.
- Remove political or religious propaganda if encountered in copied content.

## Card content modes

The app supports a single canonical pre-game setting, `imageCardCount`, from 0 to 25:

- `0`: words-only board; all 25 cards come from the selected wordpack.
- `25`: images-only board; all 25 cards come from the local picture catalog.
- `1..24`: mixed board; exactly `imageCardCount` cards are images and the remaining `25 - imageCardCount` cards are words.

The UI may present Words only, Images only, and Mixed presets, but the backend source of truth is the numeric image-card count. If fewer unique images or words are available than requested, the server rejects settings/start with a clear error instead of silently changing the count. All modes use deterministic server-side selection from the match seed so reconnects and restarts preserve the same board.
