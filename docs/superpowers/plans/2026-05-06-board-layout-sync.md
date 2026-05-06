# Board Layout Sync Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add strict card aspect-ratio layout controls, fix image-size select state display, and let moderators force current board layout preferences to all players in a room.

**Architecture:** Local board layout remains a client-side preference saved in `localStorage`. Moderators can emit a realtime room message containing only board-layout preference fields; the Go room hub validates moderator permissions and broadcasts a server-authored message to room clients, which apply and persist the incoming layout preferences.

**Tech Stack:** Go HTTP/WebSocket backend, Svelte 5 frontend, Vitest frontend unit tests, Go tests where existing websocket handler tests support it.

---

### Task 1: Frontend preference model and aspect helper

**Files:**
- Modify: `web/src/lib/gameplay.ts`
- Modify: `web/src/lib/gameplay.test.ts`

- [ ] Write failing Vitest cases for `strictCardAspectRatios` default/read/write and card class helper output.
- [ ] Run `pnpm --dir web test -- gameplay.test.ts` and verify the new tests fail because the field/helper do not exist.
- [ ] Add `strictCardAspectRatios` to `GameplayPreferences`, defaults, reader fallback, and helper that returns word/image aspect classes.
- [ ] Re-run the targeted Vitest file and verify it passes.

### Task 2: Frontend UI and realtime handling

**Files:**
- Modify: `web/src/pages/RoomPage.svelte`
- Modify: `web/src/lib/realtime.ts` if message type definitions require extension

- [ ] Add layout select binding fix using a local numeric select value or direct `bind:value` compatible with Svelte.
- [ ] Add checkbox for strict aspect ratios in Board layout settings.
- [ ] Add moderator-only force button that sends `{ type: 'forceBoardLayout', preferences: { boardColumnsMobile, boardColumnsDesktop, imageCardScale, strictCardAspectRatios } }`.
- [ ] Handle inbound `{ type: 'boardLayoutForced', preferences, by }` by merging those fields into local preferences and saving them.
- [ ] Run Svelte autofixer on `RoomPage.svelte` and run frontend tests/build.

### Task 3: Backend realtime message validation and broadcast

**Files:**
- Locate and modify Go websocket/realtime handler files.
- Modify/add Go tests near existing realtime tests if present.

- [ ] Write failing Go test or protocol test that a mod can broadcast board layout prefs and a non-mod cannot.
- [ ] Add inbound message struct fields and handler branch.
- [ ] Validate sender can manage lobby/moderate room.
- [ ] Clamp/sanitize layout values server-side before broadcasting.
- [ ] Run targeted Go test and all Go tests.

### Task 4: Docs and endpoint

**Files:**
- Modify: `docs/frontend-lobby-and-identity.md`

- [ ] Document strict aspect-ratio preference, image size dropdown behavior, and moderator force-sync behavior.
- [ ] Run full verification (`pnpm --dir web test`, `pnpm --dir web build`, `go test ./...`).
- [ ] Commit atomic changes and push current branch.
