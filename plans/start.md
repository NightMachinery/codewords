# Codewords Implementation Start Guide

This directory is the authoritative handoff for building the greenfield Codewords app, a standalone SecretCodes-style realtime word/picture deduction game.

## How to start

1. Read **every Markdown file in `plans/`** before writing code.
2. Implement in the order in `implementation-roadmap.md`.
3. Keep the project self-contained: no Docker, no hosted services, no captcha, no external fonts/CDNs/assets.
4. Use the server's installed Go version (`go1.22.2` observed on 2026-05-01); do not download or upgrade Go.
5. Use pnpm for frontend package management and keep lockfiles deterministic.
6. Commit working changes in atomic groups. Include docs updates with behavior changes.

## Plan files

- `product-spec.md` — user-facing goals, supported gameplay, language/content policy, and intranet requirements.
- `tech-stack.md` — exact technology choices and rejected alternatives.
- `architecture.md` — repo shape, runtime topology, server/static split, and data flow.
- `game-rules.md` — complete game-state and move semantics.
- `identity-and-security.md` — auth token, display name, host controls, spectators, and migrate-device links.
- `api-and-realtime.md` — HTTP and WebSocket contracts.
- `storage.md` — SQLite schema, migrations, and persistence rules.
- `frontend-spec.md` — Svelte/Tailwind UI, routes, preferences, clipboard, and WebSocket behavior.
- `assets-and-wordpacks.md` — local assets, copied wordpacks, and picture-card assets.
- `self-hosting-spec.md` — `docs/self-hosting.md`, `self_host.zsh`, Caddy, tmux, ports, proxy handling.
- `testing-and-acceptance.md` — required automated and manual acceptance coverage.
- `implementation-roadmap.md` — milestone order and completion criteria.
- `../docs/specs/secretcodes-reverse-spec.md` — reverse-mined source behavior and evidence from the original SecretCodes project.

## Big picture

Build a standalone app at `/home/ubuntu/base/codewords` with:

- Go backend using SQLite WAL for persistence.
- Svelte 5 + Vite 8 static SPA frontend styled with locally built Tailwind CSS.
- Caddy serving frontend static files in production.
- Go serving only API, WebSocket, health, and local dynamic image endpoints.
- tmux-managed self-hosting, no Docker.
- English-only UI, while preserving all copied wordpacks including non-English packs.
- Local-only/intranet-ready assets and no external service dependencies.
