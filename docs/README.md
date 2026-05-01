# Codewords Documentation

- `specs/secretcodes-reverse-spec.md` records behavior mined from `/home/ubuntu/base/FreeBoardGames.org`.
- Implementation handoff documents live in `../plans/`.
- `self-hosting.md` documents the current `self_host.zsh` lifecycle script.

## Current implementation status

Milestone 1 scaffold is implemented:

- Go backend module with `GET /healthz`.
- Svelte 5 + Vite 8 + Tailwind frontend under `web/`.
- Local asset directories under `assets/wordpacks/` and `assets/pictures/`.
- tmux/Caddy-oriented self-host skeleton in `self_host.zsh`.
