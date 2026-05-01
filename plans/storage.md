# Storage Spec

## Database

Use SQLite with:

- WAL mode enabled.
- Busy timeout enabled.
- Foreign keys enabled.
- Versioned migrations in `migrations/`.
- Application-level backups documented in self-hosting docs.

## Core tables

- `users`: internal id, auth token hash/derived id, display name, timestamps.
- `rooms`: room id, host user id, status, settings JSON, current match id, created/updated timestamps.
- `room_players`: room id, user id, team, role flags, joined timestamp, last seen timestamp.
- `migrate_links`: room id, user id, migrate id hash, created timestamp, last used timestamp, optional revoked/expired fields.
- `matches`: match id, room id, initial seed, started timestamp, finished timestamp, winner, settings JSON.
- `game_events`: match id, sequence, actor user id, event type, payload JSON, created timestamp.
- `game_snapshots`: match id, latest sequence, full authoritative state JSON, updated timestamp.
- `chat_messages`: room id, match id nullable, sender user id nullable, display name, message body, created timestamp.
- `wordpacks`: optional metadata cache for bundled wordpack files.

## Persistence rules

- Authoritative game state is derived by applying validated commands and then persisted as an event plus latest snapshot.
- On server restart, restore active rooms/matches from latest snapshots.
- Chat persists across reconnects for the room/match.
- Auth tokens and migrate ids are never stored raw.

## Indexing

Add indexes for:

- User token hash lookup.
- Room id lookup.
- Room players by room and user.
- Migrate link lookup by room and hash.
- Game events by match and sequence.
- Chat messages by room and timestamp.
