# Room Garbage Collection

Codewords does not currently garbage-collect rooms.

## Persistence lifetime

Rooms, room players, matches, snapshots, events, chat messages, and migrate links are stored in SQLite and remain there indefinitely. There is no background cleanup job, TTL setting, scheduled deletion, or automatic pruning of old rooms in the current server.

The only expiry-like behavior today is migrate-link validation: `migrate_links.expires_at` exists in the schema and is checked when resolving a migrate id, but the application does not set a default expiry for newly created migrate links.

## In-memory lifetime

The Go server also keeps a per-room runtime in memory after a room is first loaded or created. That runtime remains in the process map until the server process exits. WebSocket disconnects remove client connections from the runtime, but they do not remove the room runtime itself.

## Practical answer

A room remains available until one of these happens:

- the SQLite database is manually changed or deleted;
- a future cleanup feature deletes it;
- the server is pointed at a different database.

Restarting the server clears only in-memory room runtime objects. Persisted rooms are restored from SQLite on demand.
