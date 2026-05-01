# Storage and Identity

Milestone 3 added local SQLite persistence and server-side identity primitives. Milestone 4 now exposes these primitives through REST and WebSocket backend routes.

## SQLite

`internal/storage` opens the configured database path, enables SQLite WAL mode, foreign keys, and a 5 second busy timeout, then applies SQL migrations from `migrations/` idempotently. The default development database path is `./data/codewords.sqlite`; override it with `CODEWORDS_DATABASE_PATH`.

The initial schema stores users, rooms, room players, room-scoped migrate links, matches, ordered game events, latest authoritative snapshots, chat messages, and optional wordpack metadata. Match snapshots store concrete state JSON so future asset or wordpack changes do not rewrite existing matches.

## Identities

`internal/identity` accepts raw browser auth tokens only at the service boundary. It hashes auth tokens and room migrate ids with HMAC-SHA256 before calling storage, so SQLite never stores raw browser tokens or raw migrate ids.

Display names are validated as short plain text before persistence. Room migrate ids are random URL-safe tokens scoped to a single room and resolve only through `(roomID, migrateIDHash)`.

## Current boundary

The persistence and identity packages are now wired into the backend API. Frontend identity flows remain part of Milestone 5, while chat and picture-card behavior remain part of Milestone 7.
