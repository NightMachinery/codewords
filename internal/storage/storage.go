// Package storage provides SQLite persistence for Codewords rooms, identities, and matches.
package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

var ErrNotFound = errors.New("not found")

const RoomStatusLobby = "lobby"
const RoomStatusActive = "active"

// DB wraps the SQLite connection pool.
type DB struct{ db *sql.DB }

type User struct {
	ID          string
	TokenHash   string
	DisplayName string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Room struct {
	ID             string
	HostUserID     string
	Status         string
	SettingsJSON   string
	CurrentMatchID string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type CreateRoomParams struct {
	ID           string
	HostUserID   string
	SettingsJSON string
}

type RoomPlayer struct {
	RoomID         string
	UserID         string
	Team           string
	Spymaster      bool
	Representative bool
	Mod            bool
	JoinedAt       time.Time
	LastSeenAt     time.Time
}

type CreateMatchParams struct {
	ID           string
	RoomID       string
	Seed         int64
	SettingsJSON string
}

type Match struct {
	ID           string
	RoomID       string
	Seed         int64
	StartedAt    time.Time
	FinishedAt   time.Time
	Winner       string
	SettingsJSON string
}

type AppendGameEventParams struct {
	MatchID     string
	ActorUserID string
	EventType   string
	PayloadJSON string
}

type GameEvent struct {
	MatchID     string
	Sequence    int
	ActorUserID string
	EventType   string
	PayloadJSON string
	CreatedAt   time.Time
}

type SaveSnapshotParams struct {
	MatchID        string
	LatestSequence int
	StateJSON      string
}

type GameSnapshot struct {
	MatchID        string
	LatestSequence int
	StateJSON      string
	UpdatedAt      time.Time
}

type AddChatMessageParams struct {
	RoomID       string
	MatchID      string
	SenderUserID string
	DisplayName  string
	Body         string
}

type ChatMessage struct {
	ID           string
	RoomID       string
	MatchID      string
	SenderUserID string
	DisplayName  string
	Body         string
	CreatedAt    time.Time
}

type MigrateLink struct {
	ID            string
	RoomID        string
	UserID        string
	MigrateIDHash string
	CreatedAt     time.Time
	LastUsedAt    time.Time
}

// Open opens a SQLite database, configures required pragmas, and applies migrations.
func Open(ctx context.Context, path string) (*DB, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("open sqlite: empty path")
	}
	if err := ensureDir(path); err != nil {
		return nil, err
	}
	sqlDB, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db := &DB{db: sqlDB}
	if err := db.configure(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	if err := db.migrate(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	return db, nil
}

func ensureDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create database directory: %w", err)
	}
	return nil
}

func (d *DB) SQL() *sql.DB { return d.db }

func (d *DB) Close() error { return d.db.Close() }

func (d *DB) configure(ctx context.Context) error {
	for _, stmt := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA foreign_keys=ON",
		"PRAGMA busy_timeout=5000",
	} {
		if _, err := d.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("configure sqlite %q: %w", stmt, err)
		}
	}
	return nil
}

func findMigrationsDir() (string, error) {
	for _, dir := range []string{"migrations", "../../migrations"} {
		entries, err := os.ReadDir(dir)
		if err == nil && len(entries) > 0 {
			return dir, nil
		}
	}
	return "", fmt.Errorf("migrations directory not found")
}

func (d *DB) migrate(ctx context.Context) error {
	migrationsDir, err := findMigrationsDir()
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}
	for i, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		version := i + 1
		var exists int
		if err := d.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='schema_migrations'").Scan(&exists); err != nil {
			return fmt.Errorf("check schema_migrations: %w", err)
		}
		if exists == 1 {
			var applied int
			if err := d.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations WHERE version = ?", version).Scan(&applied); err != nil {
				return fmt.Errorf("check migration %d: %w", version, err)
			}
			if applied > 0 {
				continue
			}
		}
		content, err := os.ReadFile(filepath.Join(migrationsDir, entry.Name()))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}
		tx, err := d.db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", entry.Name(), err)
		}
		if _, err := tx.ExecContext(ctx, string(content)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", entry.Name(), err)
		}
		if _, err := tx.ExecContext(ctx, "INSERT OR IGNORE INTO schema_migrations(version) VALUES (?)", version); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %s: %w", entry.Name(), err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", entry.Name(), err)
		}
	}
	return nil
}

func (d *DB) UpsertUserByTokenHash(ctx context.Context, tokenHash, displayName string) (User, error) {
	if tokenHash == "" {
		return User{}, fmt.Errorf("token hash required")
	}
	id := uuid.NewString()
	_, err := d.db.ExecContext(ctx, `
INSERT INTO users(id, token_hash, display_name) VALUES (?, ?, ?)
ON CONFLICT(token_hash) DO UPDATE SET
  display_name = CASE WHEN excluded.display_name != '' THEN excluded.display_name ELSE users.display_name END,
  updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')`, id, tokenHash, displayName)
	if err != nil {
		return User{}, fmt.Errorf("upsert user: %w", err)
	}
	return d.UserByTokenHash(ctx, tokenHash)
}

func (d *DB) UserByTokenHash(ctx context.Context, tokenHash string) (User, error) {
	return scanUser(d.db.QueryRowContext(ctx, `SELECT id, token_hash, display_name, created_at, updated_at FROM users WHERE token_hash = ?`, tokenHash))
}

func (d *DB) UserByID(ctx context.Context, id string) (User, error) {
	return scanUser(d.db.QueryRowContext(ctx, `SELECT id, token_hash, display_name, created_at, updated_at FROM users WHERE id = ?`, id))
}

func (d *DB) UpdateDisplayName(ctx context.Context, userID, displayName string) error {
	res, err := d.db.ExecContext(ctx, `UPDATE users SET display_name = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now') WHERE id = ?`, displayName, userID)
	if err != nil {
		return fmt.Errorf("update display name: %w", err)
	}
	return requireAffected(res)
}

func (d *DB) CreateRoom(ctx context.Context, p CreateRoomParams) (Room, error) {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	if p.SettingsJSON == "" {
		p.SettingsJSON = "{}"
	}
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return Room{}, fmt.Errorf("begin create room: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO rooms(id, host_user_id, status, settings_json) VALUES (?, ?, ?, ?)`, p.ID, p.HostUserID, RoomStatusLobby, p.SettingsJSON); err != nil {
		_ = tx.Rollback()
		return Room{}, fmt.Errorf("insert room: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO room_players(room_id, user_id, mod) VALUES (?, ?, 1)`, p.ID, p.HostUserID); err != nil {
		_ = tx.Rollback()
		return Room{}, fmt.Errorf("insert host membership: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return Room{}, fmt.Errorf("commit create room: %w", err)
	}
	return d.RoomByID(ctx, p.ID)
}

func (d *DB) RoomByID(ctx context.Context, id string) (Room, error) {
	row := d.db.QueryRowContext(ctx, `SELECT id, host_user_id, status, settings_json, COALESCE(current_match_id, ''), created_at, updated_at FROM rooms WHERE id = ?`, id)
	var r Room
	var created, updated string
	if err := row.Scan(&r.ID, &r.HostUserID, &r.Status, &r.SettingsJSON, &r.CurrentMatchID, &created, &updated); err != nil {
		return Room{}, mapScanErr(err)
	}
	r.CreatedAt = parseTime(created)
	r.UpdatedAt = parseTime(updated)
	return r, nil
}

func (d *DB) SetRoomCurrentMatch(ctx context.Context, roomID, matchID string) error {
	res, err := d.db.ExecContext(ctx, `UPDATE rooms SET current_match_id = ?, status = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now') WHERE id = ?`, matchID, RoomStatusActive, roomID)
	if err != nil {
		return fmt.Errorf("set current match: %w", err)
	}
	return requireAffected(res)
}

func (d *DB) UpdateRoomSettings(ctx context.Context, roomID, settingsJSON string) error {
	res, err := d.db.ExecContext(ctx, `UPDATE rooms SET settings_json = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now') WHERE id = ?`, settingsJSON, roomID)
	if err != nil {
		return fmt.Errorf("update room settings: %w", err)
	}
	return requireAffected(res)
}

func (d *DB) UpsertRoomPlayer(ctx context.Context, p RoomPlayer) error {
	_, err := d.db.ExecContext(ctx, `
INSERT INTO room_players(room_id, user_id, team, spymaster, representative, mod) VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(room_id, user_id) DO UPDATE SET team = excluded.team, spymaster = excluded.spymaster, representative = excluded.representative, mod = excluded.mod, last_seen_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')`, p.RoomID, p.UserID, p.Team, boolInt(p.Spymaster), boolInt(p.Representative), boolInt(p.Mod))
	if err != nil {
		return fmt.Errorf("upsert room player: %w", err)
	}
	return nil
}

func (d *DB) RoomPlayers(ctx context.Context, roomID string) ([]RoomPlayer, error) {
	rows, err := d.db.QueryContext(ctx, `SELECT room_id, user_id, team, spymaster, representative, mod, joined_at, last_seen_at FROM room_players WHERE room_id = ? ORDER BY joined_at, user_id`, roomID)
	if err != nil {
		return nil, fmt.Errorf("query room players: %w", err)
	}
	defer rows.Close()
	var players []RoomPlayer
	for rows.Next() {
		var p RoomPlayer
		var spy, rep, mod int
		var joined, seen string
		if err := rows.Scan(&p.RoomID, &p.UserID, &p.Team, &spy, &rep, &mod, &joined, &seen); err != nil {
			return nil, fmt.Errorf("scan room player: %w", err)
		}
		p.Spymaster = spy == 1
		p.Representative = rep == 1
		p.Mod = mod == 1
		p.JoinedAt = parseTime(joined)
		p.LastSeenAt = parseTime(seen)
		players = append(players, p)
	}
	return players, rows.Err()
}

func (d *DB) CreateMatch(ctx context.Context, p CreateMatchParams) (Match, error) {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	_, err := d.db.ExecContext(ctx, `INSERT INTO matches(id, room_id, initial_seed, settings_json) VALUES (?, ?, ?, ?)`, p.ID, p.RoomID, p.Seed, p.SettingsJSON)
	if err != nil {
		return Match{}, fmt.Errorf("create match: %w", err)
	}
	return d.MatchByID(ctx, p.ID)
}

func (d *DB) MatchByID(ctx context.Context, id string) (Match, error) {
	row := d.db.QueryRowContext(ctx, `SELECT id, room_id, initial_seed, started_at, COALESCE(finished_at, ''), winner, settings_json FROM matches WHERE id = ?`, id)
	var m Match
	var started, finished string
	if err := row.Scan(&m.ID, &m.RoomID, &m.Seed, &started, &finished, &m.Winner, &m.SettingsJSON); err != nil {
		return Match{}, mapScanErr(err)
	}
	m.StartedAt = parseTime(started)
	m.FinishedAt = parseTime(finished)
	return m, nil
}

func (d *DB) AppendGameEvent(ctx context.Context, p AppendGameEventParams) (GameEvent, error) {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return GameEvent{}, fmt.Errorf("begin append event: %w", err)
	}
	var seq int
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(sequence), 0) + 1 FROM game_events WHERE match_id = ?`, p.MatchID).Scan(&seq); err != nil {
		_ = tx.Rollback()
		return GameEvent{}, fmt.Errorf("next event sequence: %w", err)
	}
	actor := nullString(p.ActorUserID)
	if _, err := tx.ExecContext(ctx, `INSERT INTO game_events(match_id, sequence, actor_user_id, event_type, payload_json) VALUES (?, ?, ?, ?, ?)`, p.MatchID, seq, actor, p.EventType, p.PayloadJSON); err != nil {
		_ = tx.Rollback()
		return GameEvent{}, fmt.Errorf("insert event: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return GameEvent{}, fmt.Errorf("commit event: %w", err)
	}
	return d.GameEvent(ctx, p.MatchID, seq)
}

func (d *DB) GameEvent(ctx context.Context, matchID string, seq int) (GameEvent, error) {
	row := d.db.QueryRowContext(ctx, `SELECT match_id, sequence, COALESCE(actor_user_id, ''), event_type, payload_json, created_at FROM game_events WHERE match_id = ? AND sequence = ?`, matchID, seq)
	var e GameEvent
	var created string
	if err := row.Scan(&e.MatchID, &e.Sequence, &e.ActorUserID, &e.EventType, &e.PayloadJSON, &created); err != nil {
		return GameEvent{}, mapScanErr(err)
	}
	e.CreatedAt = parseTime(created)
	return e, nil
}

func (d *DB) SaveSnapshot(ctx context.Context, p SaveSnapshotParams) error {
	_, err := d.db.ExecContext(ctx, `INSERT INTO game_snapshots(match_id, latest_sequence, state_json) VALUES (?, ?, ?) ON CONFLICT(match_id) DO UPDATE SET latest_sequence = excluded.latest_sequence, state_json = excluded.state_json, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')`, p.MatchID, p.LatestSequence, p.StateJSON)
	if err != nil {
		return fmt.Errorf("save snapshot: %w", err)
	}
	return nil
}

func (d *DB) LatestSnapshot(ctx context.Context, matchID string) (GameSnapshot, error) {
	row := d.db.QueryRowContext(ctx, `SELECT match_id, latest_sequence, state_json, updated_at FROM game_snapshots WHERE match_id = ?`, matchID)
	var s GameSnapshot
	var updated string
	if err := row.Scan(&s.MatchID, &s.LatestSequence, &s.StateJSON, &updated); err != nil {
		return GameSnapshot{}, mapScanErr(err)
	}
	s.UpdatedAt = parseTime(updated)
	return s, nil
}

func (d *DB) AddChatMessage(ctx context.Context, p AddChatMessageParams) (ChatMessage, error) {
	id := uuid.NewString()
	_, err := d.db.ExecContext(ctx, `INSERT INTO chat_messages(id, room_id, match_id, sender_user_id, display_name, body) VALUES (?, ?, ?, ?, ?, ?)`, id, p.RoomID, nullString(p.MatchID), nullString(p.SenderUserID), p.DisplayName, p.Body)
	if err != nil {
		return ChatMessage{}, fmt.Errorf("add chat message: %w", err)
	}
	return d.chatMessage(ctx, id)
}

func (d *DB) chatMessage(ctx context.Context, id string) (ChatMessage, error) {
	row := d.db.QueryRowContext(ctx, `SELECT id, room_id, COALESCE(match_id, ''), COALESCE(sender_user_id, ''), display_name, body, created_at FROM chat_messages WHERE id = ?`, id)
	var m ChatMessage
	var created string
	if err := row.Scan(&m.ID, &m.RoomID, &m.MatchID, &m.SenderUserID, &m.DisplayName, &m.Body, &created); err != nil {
		return ChatMessage{}, mapScanErr(err)
	}
	m.CreatedAt = parseTime(created)
	return m, nil
}

func (d *DB) ChatMessages(ctx context.Context, roomID string, limit int) ([]ChatMessage, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := d.db.QueryContext(ctx, `SELECT id, room_id, COALESCE(match_id, ''), COALESCE(sender_user_id, ''), display_name, body, created_at FROM chat_messages WHERE room_id = ? ORDER BY created_at, id LIMIT ?`, roomID, limit)
	if err != nil {
		return nil, fmt.Errorf("query chat messages: %w", err)
	}
	defer rows.Close()
	var messages []ChatMessage
	for rows.Next() {
		var m ChatMessage
		var created string
		if err := rows.Scan(&m.ID, &m.RoomID, &m.MatchID, &m.SenderUserID, &m.DisplayName, &m.Body, &created); err != nil {
			return nil, fmt.Errorf("scan chat message: %w", err)
		}
		m.CreatedAt = parseTime(created)
		messages = append(messages, m)
	}
	return messages, rows.Err()
}

func (d *DB) UpsertMigrateLink(ctx context.Context, link MigrateLink) (MigrateLink, error) {
	if link.ID == "" {
		link.ID = uuid.NewString()
	}
	_, err := d.db.ExecContext(ctx, `
INSERT INTO migrate_links(id, room_id, user_id, migrate_id_hash) VALUES (?, ?, ?, ?)
ON CONFLICT(room_id, user_id) DO UPDATE SET migrate_id_hash = excluded.migrate_id_hash, last_used_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now'), revoked_at = NULL`, link.ID, link.RoomID, link.UserID, link.MigrateIDHash)
	if err != nil {
		return MigrateLink{}, fmt.Errorf("upsert migrate link: %w", err)
	}
	return d.ResolveMigrateLink(ctx, link.RoomID, link.MigrateIDHash)
}

func (d *DB) ResolveMigrateLink(ctx context.Context, roomID, migrateIDHash string) (MigrateLink, error) {
	row := d.db.QueryRowContext(ctx, `SELECT id, room_id, user_id, migrate_id_hash, created_at, COALESCE(last_used_at, '') FROM migrate_links WHERE room_id = ? AND migrate_id_hash = ? AND revoked_at IS NULL AND (expires_at IS NULL OR expires_at > strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))`, roomID, migrateIDHash)
	var l MigrateLink
	var created, used string
	if err := row.Scan(&l.ID, &l.RoomID, &l.UserID, &l.MigrateIDHash, &created, &used); err != nil {
		return MigrateLink{}, mapScanErr(err)
	}
	l.CreatedAt = parseTime(created)
	l.LastUsedAt = parseTime(used)
	return l, nil
}

type rowScanner interface{ Scan(dest ...any) error }

func scanUser(row rowScanner) (User, error) {
	var u User
	var created, updated string
	if err := row.Scan(&u.ID, &u.TokenHash, &u.DisplayName, &created, &updated); err != nil {
		return User{}, mapScanErr(err)
	}
	u.CreatedAt = parseTime(created)
	u.UpdatedAt = parseTime(updated)
	return u, nil
}

func mapScanErr(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func requireAffected(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
func nullString(v string) sql.NullString { return sql.NullString{String: v, Valid: v != ""} }
func parseTime(value string) time.Time   { t, _ := time.Parse(time.RFC3339Nano, value); return t }
