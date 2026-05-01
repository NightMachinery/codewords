package storage_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/NightMachinery/codewords/internal/storage"
)

func TestOpenAppliesMigrationsAndEnablesSQLitePragmas(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	defer db.Close()

	if got := querySingleString(t, db.SQL(), "PRAGMA journal_mode"); got != "wal" {
		t.Fatalf("expected WAL journal mode, got %q", got)
	}
	if got := querySingleInt(t, db.SQL(), "PRAGMA foreign_keys"); got != 1 {
		t.Fatalf("expected foreign keys enabled, got %d", got)
	}
	if got := querySingleInt(t, db.SQL(), "PRAGMA busy_timeout"); got <= 0 {
		t.Fatalf("expected busy timeout enabled, got %d", got)
	}

	// Reopening the same database proves migrations are idempotent.
	path := filepath.Join(t.TempDir(), "idempotent.sqlite")
	first, err := storage.Open(ctx, path)
	if err != nil {
		t.Fatalf("open first db: %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("close first db: %v", err)
	}
	second, err := storage.Open(ctx, path)
	if err != nil {
		t.Fatalf("open migrated db again: %v", err)
	}
	defer second.Close()
	if got := querySingleInt(t, second.SQL(), "SELECT COUNT(*) FROM schema_migrations"); got < 1 {
		t.Fatalf("expected applied migrations, got %d", got)
	}
}

func TestUsersRoomsMatchesSnapshotsEventsAndChatPersist(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	defer db.Close()

	user, err := db.UpsertUserByTokenHash(ctx, "hash-user", "Alice")
	if err != nil {
		t.Fatalf("upsert user: %v", err)
	}
	again, err := db.UpsertUserByTokenHash(ctx, "hash-user", "")
	if err != nil {
		t.Fatalf("fetch user: %v", err)
	}
	if again.ID != user.ID || again.DisplayName != "Alice" {
		t.Fatalf("expected persisted user/display name, got %#v then %#v", user, again)
	}
	if err := db.UpdateDisplayName(ctx, user.ID, "Alice Updated"); err != nil {
		t.Fatalf("update display name: %v", err)
	}
	updated, err := db.UserByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("user by id: %v", err)
	}
	if updated.DisplayName != "Alice Updated" {
		t.Fatalf("expected updated display name, got %q", updated.DisplayName)
	}

	room, err := db.CreateRoom(ctx, storage.CreateRoomParams{ID: "room-1", HostUserID: user.ID, SettingsJSON: `{"wordpackId":"english"}`})
	if err != nil {
		t.Fatalf("create room: %v", err)
	}
	if room.HostUserID != user.ID || room.Status != storage.RoomStatusLobby {
		t.Fatalf("unexpected room: %#v", room)
	}
	players, err := db.RoomPlayers(ctx, room.ID)
	if err != nil {
		t.Fatalf("room players: %v", err)
	}
	if len(players) != 1 || players[0].UserID != user.ID {
		t.Fatalf("expected host membership, got %#v", players)
	}
	if err := db.UpsertRoomPlayer(ctx, storage.RoomPlayer{RoomID: room.ID, UserID: user.ID, Team: "blue", Spymaster: true}); err != nil {
		t.Fatalf("upsert room player: %v", err)
	}
	players, err = db.RoomPlayers(ctx, room.ID)
	if err != nil {
		t.Fatalf("room players after update: %v", err)
	}
	if players[0].Team != "blue" || !players[0].Spymaster {
		t.Fatalf("expected updated role fields, got %#v", players[0])
	}

	match, err := db.CreateMatch(ctx, storage.CreateMatchParams{ID: "match-1", RoomID: room.ID, Seed: 99, SettingsJSON: `{"seed":99}`})
	if err != nil {
		t.Fatalf("create match: %v", err)
	}
	if err := db.SetRoomCurrentMatch(ctx, room.ID, match.ID); err != nil {
		t.Fatalf("set current match: %v", err)
	}
	event, err := db.AppendGameEvent(ctx, storage.AppendGameEventParams{MatchID: match.ID, ActorUserID: user.ID, EventType: "match_started", PayloadJSON: `{"ok":true}`})
	if err != nil {
		t.Fatalf("append event: %v", err)
	}
	if event.Sequence != 1 {
		t.Fatalf("expected first sequence 1, got %d", event.Sequence)
	}
	second, err := db.AppendGameEvent(ctx, storage.AppendGameEventParams{MatchID: match.ID, ActorUserID: user.ID, EventType: "guess_accepted", PayloadJSON: `{"index":0}`})
	if err != nil {
		t.Fatalf("append second event: %v", err)
	}
	if second.Sequence != 2 {
		t.Fatalf("expected second sequence 2, got %d", second.Sequence)
	}
	if err := db.SaveSnapshot(ctx, storage.SaveSnapshotParams{MatchID: match.ID, LatestSequence: second.Sequence, StateJSON: `{"phase":"active"}`}); err != nil {
		t.Fatalf("save snapshot: %v", err)
	}
	snapshot, err := db.LatestSnapshot(ctx, match.ID)
	if err != nil {
		t.Fatalf("latest snapshot: %v", err)
	}
	if snapshot.LatestSequence != 2 || snapshot.StateJSON != `{"phase":"active"}` {
		t.Fatalf("unexpected snapshot: %#v", snapshot)
	}

	if _, err := db.AddChatMessage(ctx, storage.AddChatMessageParams{RoomID: room.ID, MatchID: match.ID, SenderUserID: user.ID, DisplayName: "Alice", Body: "hello"}); err != nil {
		t.Fatalf("add chat: %v", err)
	}
	if _, err := db.AddChatMessage(ctx, storage.AddChatMessageParams{RoomID: room.ID, DisplayName: "Spectator", Body: "read-only soon"}); err != nil {
		t.Fatalf("add spectator chat: %v", err)
	}
	messages, err := db.ChatMessages(ctx, room.ID, 10)
	if err != nil {
		t.Fatalf("chat messages: %v", err)
	}
	if len(messages) != 2 || messages[0].Body != "hello" || messages[1].Body != "read-only soon" {
		t.Fatalf("expected ordered messages, got %#v", messages)
	}
}

func TestMigrateLinksAreHashedReusableAndRoomScoped(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	defer db.Close()
	user, err := db.UpsertUserByTokenHash(ctx, "hash-user", "Alice")
	if err != nil {
		t.Fatalf("upsert user: %v", err)
	}
	other, err := db.UpsertUserByTokenHash(ctx, "hash-other", "Bob")
	if err != nil {
		t.Fatalf("upsert other user: %v", err)
	}
	if _, err := db.CreateRoom(ctx, storage.CreateRoomParams{ID: "room-a", HostUserID: user.ID, SettingsJSON: `{}`}); err != nil {
		t.Fatalf("create room a: %v", err)
	}
	if _, err := db.CreateRoom(ctx, storage.CreateRoomParams{ID: "room-b", HostUserID: other.ID, SettingsJSON: `{}`}); err != nil {
		t.Fatalf("create room b: %v", err)
	}

	first, err := db.UpsertMigrateLink(ctx, storage.MigrateLink{RoomID: "room-a", UserID: user.ID, MigrateIDHash: "hash-migrate"})
	if err != nil {
		t.Fatalf("upsert migrate link: %v", err)
	}
	if first.MigrateIDHash == "raw-token" {
		t.Fatalf("raw migrate id should not be stored")
	}
	time.Sleep(time.Millisecond)
	reused, err := db.UpsertMigrateLink(ctx, storage.MigrateLink{RoomID: "room-a", UserID: user.ID, MigrateIDHash: "hash-migrate"})
	if err != nil {
		t.Fatalf("reuse migrate link: %v", err)
	}
	if reused.ID != first.ID || reused.LastUsedAt.IsZero() {
		t.Fatalf("expected reused link with last-used timestamp, got first=%#v reused=%#v", first, reused)
	}
	resolved, err := db.ResolveMigrateLink(ctx, "room-a", "hash-migrate")
	if err != nil {
		t.Fatalf("resolve migrate link: %v", err)
	}
	if resolved.UserID != user.ID {
		t.Fatalf("expected room-a user %q, got %#v", user.ID, resolved)
	}
	if _, err := db.ResolveMigrateLink(ctx, "room-b", "hash-migrate"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected room scoped miss, got %v", err)
	}
}

func openTestDB(t *testing.T) *storage.DB {
	t.Helper()
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "codewords.sqlite"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	return db
}

func querySingleString(t *testing.T, db *sql.DB, query string) string {
	t.Helper()
	var value string
	if err := db.QueryRow(query).Scan(&value); err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	return value
}

func querySingleInt(t *testing.T, db *sql.DB, query string) int {
	t.Helper()
	var value int
	if err := db.QueryRow(query).Scan(&value); err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	return value
}
