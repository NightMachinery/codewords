package config

import "testing"

func TestFromEnvUsesDefaults(t *testing.T) {
	t.Setenv("CODEWORDS_ADDR", "")
	t.Setenv("CODEWORDS_DATABASE_PATH", "")

	cfg := FromEnv()

	if cfg.Addr != defaultAddr {
		t.Fatalf("expected default addr %q, got %q", defaultAddr, cfg.Addr)
	}
	if cfg.DatabasePath != defaultDatabasePath {
		t.Fatalf("expected default database path %q, got %q", defaultDatabasePath, cfg.DatabasePath)
	}
}

func TestFromEnvOverridesValues(t *testing.T) {
	t.Setenv("CODEWORDS_ADDR", "0.0.0.0:9999")
	t.Setenv("CODEWORDS_DATABASE_PATH", "/tmp/codewords.sqlite")

	cfg := FromEnv()

	if cfg.Addr != "0.0.0.0:9999" {
		t.Fatalf("expected env addr override, got %q", cfg.Addr)
	}
	if cfg.DatabasePath != "/tmp/codewords.sqlite" {
		t.Fatalf("expected env database path override, got %q", cfg.DatabasePath)
	}
}
