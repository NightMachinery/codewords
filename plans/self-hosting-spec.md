# Self-Hosting Spec

## Files to implement

- `docs/self-hosting.md`
- `self_host.zsh`

## Commands

`self_host.zsh` must support:

- `setup [url]`: stop existing sessions, install/setup dependencies, build, update Caddy config, start production.
- `redeploy [url]`: stop existing sessions, rebuild latest local changes, update Caddy config, start production.
- `start [url]`: stop prod/dev sessions, check ports, start production tmux session(s).
- `stop`: stop all Codewords prod/dev tmux sessions.
- `dev-start [url]`: stop prod/dev sessions, check ports, start development tmux sessions with hot reload.

Default URL: `http://codewords.pinky.lilf.ir`.

## Process management

Use this helper style:

```zsh
tmuxnew () {
  tmux kill-session -t "$1" &> /dev/null || true
  tmux new -d -s "$@"
}
```

Use hardened tmux env syntax when passing env vars:

```zsh
tmux new -d -s session-name -e "VARIABLE=value" -- command
```

Pass through existing proxy env vars when present. Do not hardcode proxy host or port into the script.

## Node/pnpm in zsh

When Node is needed in zsh:

```zsh
nvm-load
nvm use VERSION
```

Use pnpm, not npm/yarn. pnpm must use the lockfile and deduplicate to avoid wasting disk.

## Caddy

- Update `~/Caddyfile` with a managed Codewords block.
- Caddy serves `web/dist` static files directly in production.
- Caddy reverse-proxies backend paths to Go:
  - `/api/*`
  - `/ws/*`
  - `/healthz`
  - picture dynamic endpoints if any.
- Include both HTTP-compatible behavior and compression.
- Do not assume HTTPS is available; site must work on HTTP.
- Reload Caddy safely after updating config.

## Ports

- Choose fixed local backend and dev ports and document them.
- Before start/dev-start/redeploy/setup, check required ports are not already in use by unrelated processes.
- If occupied by an old Codewords tmux session, stop it first.
- If still occupied, fail with a clear error.

## Production build

- Build frontend to `web/dist`.
- Build Go server binary.
- Start only the Go backend in tmux; Caddy serves static files.

## Development mode

- Start Go dev process and Vite dev server in tmux.
- Prefer hot reload where simple; otherwise document manual restart.
- Caddy may proxy to Vite for frontend while still proxying backend paths to Go.

## docs/self-hosting.md contents

Document:

- Requirements: Caddy, tmux, Go, Node via nvm, pnpm.
- Setup command and default/custom URL.
- Production lifecycle commands.
- Development lifecycle command.
- Data directory and backup/restore notes.
- Proxy environment variable behavior.
- No Docker requirement.
