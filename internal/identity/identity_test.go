package identity_test

import (
	"context"
	"strings"
	"testing"

	"github.com/NightMachinery/codewords/internal/identity"
)

func TestServiceBootstrapsIdentityWithoutStoringRawToken(t *testing.T) {
	ctx := context.Background()
	store := identity.NewMemoryStore()
	svc := identity.NewService(store, identity.Options{HashKey: []byte("test-key")})

	first, err := svc.Bootstrap(ctx, "raw-browser-token", "")
	if err != nil {
		t.Fatalf("bootstrap first: %v", err)
	}
	if first.UserID == "" || first.AuthTokenHash == "" {
		t.Fatalf("expected user id and hash, got %#v", first)
	}
	if strings.Contains(first.AuthTokenHash, "raw-browser-token") {
		t.Fatalf("hash must not contain raw token: %q", first.AuthTokenHash)
	}

	if err := svc.SaveDisplayName(ctx, first.UserID, "Alice"); err != nil {
		t.Fatalf("save display name: %v", err)
	}
	second, err := svc.Bootstrap(ctx, "raw-browser-token", "")
	if err != nil {
		t.Fatalf("bootstrap second: %v", err)
	}
	if second.UserID != first.UserID || second.DisplayName != "Alice" {
		t.Fatalf("expected same user with saved name, got first=%#v second=%#v", first, second)
	}
}

func TestServiceRejectsEmptyTokensAndDisplayNames(t *testing.T) {
	svc := identity.NewService(identity.NewMemoryStore(), identity.Options{HashKey: []byte("test-key")})
	if _, err := svc.Bootstrap(context.Background(), "   ", ""); err == nil {
		t.Fatalf("expected empty auth token rejection")
	}
	created, err := svc.Bootstrap(context.Background(), "token", "")
	if err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	if err := svc.SaveDisplayName(context.Background(), created.UserID, strings.Repeat("x", 81)); err == nil {
		t.Fatalf("expected long display name rejection")
	}
	if err := svc.SaveDisplayName(context.Background(), created.UserID, "<script>alert(1)</script>"); err == nil {
		t.Fatalf("expected display name with angle brackets rejected")
	}
}

func TestMigrateLinksAreOpaqueReusableAndRoomScoped(t *testing.T) {
	ctx := context.Background()
	store := identity.NewMemoryStore()
	svc := identity.NewService(store, identity.Options{HashKey: []byte("test-key"), MigrateIDBytes: 16})
	created, err := svc.Bootstrap(ctx, "token-a", "Alice")
	if err != nil {
		t.Fatalf("bootstrap: %v", err)
	}

	first, err := svc.CreateMigrateLink(ctx, "room-a", created.UserID)
	if err != nil {
		t.Fatalf("create first migrate link: %v", err)
	}
	if first.MigrateID == "" || first.MigrateIDHash == "" {
		t.Fatalf("expected migrate id and hash, got %#v", first)
	}
	if strings.Contains(first.MigrateIDHash, first.MigrateID) {
		t.Fatalf("hash must not contain raw migrate id")
	}
	reused, err := svc.CreateMigrateLink(ctx, "room-a", created.UserID)
	if err != nil {
		t.Fatalf("reuse migrate link: %v", err)
	}
	if reused.MigrateID != first.MigrateID || reused.UserID != created.UserID {
		t.Fatalf("expected reusable room/user migrate id, got first=%#v reused=%#v", first, reused)
	}

	resolved, err := svc.ResolveMigrate(ctx, "room-a", first.MigrateID)
	if err != nil {
		t.Fatalf("resolve migrate: %v", err)
	}
	if resolved.UserID != created.UserID || resolved.RoomID != "room-a" {
		t.Fatalf("unexpected resolve: %#v", resolved)
	}
	if _, err := svc.ResolveMigrate(ctx, "room-b", first.MigrateID); err == nil {
		t.Fatalf("expected room-scoped migrate id to fail in other room")
	}
}
