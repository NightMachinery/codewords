// Package identity manages browser identities and room-scoped migrate links.
package identity

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/NightMachinery/codewords/internal/storage"
)

const (
	defaultMigrateIDBytes = 32
	maxDisplayNameRunes   = 80
)

// Options configures identity hashing and token generation.
type Options struct {
	HashKey        []byte
	MigrateIDBytes int
}

// BootstrapResult is returned when a browser auth token maps to a user.
type BootstrapResult struct {
	UserID        string
	DisplayName   string
	AuthTokenHash string
}

// MigrateLinkResult includes the one-time/raw migrate id for the caller and its stored hash.
type MigrateLinkResult struct {
	RoomID        string
	UserID        string
	MigrateID     string
	MigrateIDHash string
}

// Store is the persistence boundary required by Service.
type Store interface {
	UpsertUserByTokenHash(context.Context, string, string) (storage.User, error)
	UpdateDisplayName(context.Context, string, string) error
	UpsertMigrateLink(context.Context, storage.MigrateLink) (storage.MigrateLink, error)
	ResolveMigrateLink(context.Context, string, string) (storage.MigrateLink, error)
}

// Service owns hashing, validation, and opaque migrate id generation.
type Service struct {
	store          Store
	hashKey        []byte
	migrateIDBytes int
	mu             sync.Mutex
	rawMigrateIDs  map[string]string
}

// NewService creates an identity service.
func NewService(store Store, opts Options) *Service {
	key := opts.HashKey
	if len(key) == 0 {
		key = []byte("codewords-development-hash-key")
	}
	bytes := opts.MigrateIDBytes
	if bytes <= 0 {
		bytes = defaultMigrateIDBytes
	}
	return &Service{store: store, hashKey: key, migrateIDBytes: bytes, rawMigrateIDs: map[string]string{}}
}

// Bootstrap creates or fetches the identity associated with a raw browser auth token.
func (s *Service) Bootstrap(ctx context.Context, rawAuthToken string, displayName string) (BootstrapResult, error) {
	token := strings.TrimSpace(rawAuthToken)
	if token == "" {
		return BootstrapResult{}, fmt.Errorf("auth token required")
	}
	name, err := validateOptionalDisplayName(displayName)
	if err != nil {
		return BootstrapResult{}, err
	}
	hash := s.hashToken("auth", token)
	user, err := s.store.UpsertUserByTokenHash(ctx, hash, name)
	if err != nil {
		return BootstrapResult{}, err
	}
	return BootstrapResult{UserID: user.ID, DisplayName: user.DisplayName, AuthTokenHash: hash}, nil
}

// SaveDisplayName validates and persists a display name for a user.
func (s *Service) SaveDisplayName(ctx context.Context, userID string, displayName string) error {
	name, err := validateRequiredDisplayName(displayName)
	if err != nil {
		return err
	}
	return s.store.UpdateDisplayName(ctx, userID, name)
}

// CreateMigrateLink creates or refreshes a room-scoped migrate id for a room/user pair.
func (s *Service) CreateMigrateLink(ctx context.Context, roomID, userID string) (MigrateLinkResult, error) {
	if strings.TrimSpace(roomID) == "" || strings.TrimSpace(userID) == "" {
		return MigrateLinkResult{}, fmt.Errorf("room id and user id required")
	}
	cacheKey := roomID + "\x00" + userID
	s.mu.Lock()
	raw := s.rawMigrateIDs[cacheKey]
	s.mu.Unlock()
	if raw == "" {
		generated, err := randomToken(s.migrateIDBytes)
		if err != nil {
			return MigrateLinkResult{}, err
		}
		raw = generated
	}
	hash := s.hashToken("migrate", raw)
	link, err := s.store.UpsertMigrateLink(ctx, storage.MigrateLink{RoomID: roomID, UserID: userID, MigrateIDHash: hash})
	if err != nil {
		return MigrateLinkResult{}, err
	}
	s.mu.Lock()
	s.rawMigrateIDs[cacheKey] = raw
	s.mu.Unlock()
	return MigrateLinkResult{RoomID: link.RoomID, UserID: link.UserID, MigrateID: raw, MigrateIDHash: hash}, nil
}

// ResolveMigrate resolves a raw migrate id only for the supplied room.
func (s *Service) ResolveMigrate(ctx context.Context, roomID, rawMigrateID string) (MigrateLinkResult, error) {
	if strings.TrimSpace(roomID) == "" || strings.TrimSpace(rawMigrateID) == "" {
		return MigrateLinkResult{}, fmt.Errorf("room id and migrate id required")
	}
	hash := s.hashToken("migrate", rawMigrateID)
	link, err := s.store.ResolveMigrateLink(ctx, roomID, hash)
	if err != nil {
		return MigrateLinkResult{}, err
	}
	return MigrateLinkResult{RoomID: link.RoomID, UserID: link.UserID, MigrateID: rawMigrateID, MigrateIDHash: hash}, nil
}

func (s *Service) hashToken(scope, raw string) string {
	mac := hmac.New(sha256.New, s.hashKey)
	_, _ = mac.Write([]byte(scope))
	_, _ = mac.Write([]byte{0})
	_, _ = mac.Write([]byte(raw))
	return hex.EncodeToString(mac.Sum(nil))
}

func randomToken(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate random token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func validateRequiredDisplayName(name string) (string, error) {
	trimmed, err := validateOptionalDisplayName(name)
	if err != nil {
		return "", err
	}
	if trimmed == "" {
		return "", fmt.Errorf("display name required")
	}
	return trimmed, nil
}

func validateOptionalDisplayName(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", nil
	}
	if !utf8.ValidString(trimmed) {
		return "", fmt.Errorf("display name must be valid utf-8")
	}
	if utf8.RuneCountInString(trimmed) > maxDisplayNameRunes {
		return "", fmt.Errorf("display name too long")
	}
	if strings.ContainsAny(trimmed, "<>") {
		return "", fmt.Errorf("display name contains unsupported characters")
	}
	return trimmed, nil
}

// MemoryStore is a test and development Store implementation.
type MemoryStore struct {
	mu            sync.Mutex
	usersByHash   map[string]storage.User
	usersByID     map[string]storage.User
	migrateByUser map[string]storage.MigrateLink
	migrateByHash map[string]storage.MigrateLink
	counter       int
}

// NewMemoryStore creates an in-memory identity store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{usersByHash: map[string]storage.User{}, usersByID: map[string]storage.User{}, migrateByUser: map[string]storage.MigrateLink{}, migrateByHash: map[string]storage.MigrateLink{}}
}

func (m *MemoryStore) UpsertUserByTokenHash(_ context.Context, tokenHash, displayName string) (storage.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if user, ok := m.usersByHash[tokenHash]; ok {
		if displayName != "" {
			user.DisplayName = displayName
			m.usersByHash[tokenHash] = user
			m.usersByID[user.ID] = user
		}
		return user, nil
	}
	m.counter++
	user := storage.User{ID: fmt.Sprintf("user-%d", m.counter), TokenHash: tokenHash, DisplayName: displayName}
	m.usersByHash[tokenHash] = user
	m.usersByID[user.ID] = user
	return user, nil
}

func (m *MemoryStore) UpdateDisplayName(_ context.Context, userID, displayName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	user, ok := m.usersByID[userID]
	if !ok {
		return storage.ErrNotFound
	}
	user.DisplayName = displayName
	m.usersByID[userID] = user
	m.usersByHash[user.TokenHash] = user
	return nil
}

func (m *MemoryStore) UpsertMigrateLink(_ context.Context, link storage.MigrateLink) (storage.MigrateLink, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := link.RoomID + "\x00" + link.UserID
	if existing, ok := m.migrateByUser[key]; ok {
		delete(m.migrateByHash, existing.RoomID+"\x00"+existing.MigrateIDHash)
		existing.MigrateIDHash = link.MigrateIDHash
		m.migrateByUser[key] = existing
		m.migrateByHash[existing.RoomID+"\x00"+existing.MigrateIDHash] = existing
		return existing, nil
	}
	m.counter++
	link.ID = fmt.Sprintf("migrate-%d", m.counter)
	m.migrateByUser[key] = link
	m.migrateByHash[link.RoomID+"\x00"+link.MigrateIDHash] = link
	return link, nil
}

func (m *MemoryStore) ResolveMigrateLink(_ context.Context, roomID, migrateIDHash string) (storage.MigrateLink, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	link, ok := m.migrateByHash[roomID+"\x00"+migrateIDHash]
	if !ok {
		return storage.MigrateLink{}, storage.ErrNotFound
	}
	return link, nil
}

var _ Store = (*MemoryStore)(nil)
var _ Store = (*storage.DB)(nil)
