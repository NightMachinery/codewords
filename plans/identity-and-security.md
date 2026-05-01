# Identity and Security

## Browser identity

- On first visit, frontend generates a high-entropy random auth token and stores it in LocalStorage.
- Server never returns or exposes another user’s real auth token.
- Store only a hash/derived identifier for auth tokens in SQLite.
- Display name is associated with this identity and persisted server-side so the user is not repeatedly prompted.
- If LocalStorage token exists, bootstrap identity silently and fetch saved display name.

## Room membership and authorization

- Room host is the identity that created the room unless explicitly transferred in future versions.
- Host-only actions: assigning teams, toggling roles for others, changing room settings, starting match, creating room-scoped migrate links for self.
- User actions: setting display name, joining a room as self, switching own team when allowed, toggling own non-host preferences.
- Game commands are accepted only from authorized active players for the current state.

## Migrate-device links

- Add a visible “Migrate device” button for seated users.
- Button requests or reuses a room-scoped opaque migrate id for `(room_id, user_identity)` and copies a room URL containing it.
- The migrate id is random, unguessable, room-specific, revocable/expirable if later desired, and stored hashed server-side if practical.
- The migrate URL never includes the real LocalStorage auth token.
- Opening a migrate URL uses the linked user identity **only for that room**.
- The migrate id remains in the URL so refresh continues to use that room identity.
- Do not overwrite the browser’s global LocalStorage auth token when consuming a migrate URL.
- Outside that room, the browser continues to use its original LocalStorage identity.

## Spectators

- Anonymous users may open an already-started or spectator-allowed room as read-only spectators.
- Spectators receive sanitized non-spymaster snapshots.
- Anonymous spectators can read chat but cannot send chat, take seats, reveal cards, pass, or toggle views.

## Security defaults

- Validate and authorize all commands server-side.
- Do not trust client role, team, or hidden card state.
- Use same-site cookies only if added later; v1 can rely on explicit auth token headers/body from LocalStorage over private hosting.
- Do not log raw auth tokens or migrate ids.
