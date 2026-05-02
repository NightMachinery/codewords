package config

import "testing"

func TestFromEnvUsesDefaults(t *testing.T) {
	t.Setenv("CODEWORDS_ADDR", "")
	t.Setenv("CODEWORDS_DATABASE_PATH", "")
	t.Setenv("CODEWORDS_PICTURES_DIR", "")

	cfg := FromEnv()

	if cfg.Addr != defaultAddr {
		t.Fatalf("expected default addr %q, got %q", defaultAddr, cfg.Addr)
	}
	if cfg.DatabasePath != defaultDatabasePath {
		t.Fatalf("expected default database path %q, got %q", defaultDatabasePath, cfg.DatabasePath)
	}
	if cfg.PicturesDir != defaultPicturesDir {
		t.Fatalf("expected default pictures dir %q, got %q", defaultPicturesDir, cfg.PicturesDir)
	}
}

func TestFromEnvOverridesValues(t *testing.T) {
	t.Setenv("CODEWORDS_ADDR", "0.0.0.0:9999")
	t.Setenv("CODEWORDS_DATABASE_PATH", "/tmp/codewords.sqlite")
	t.Setenv("CODEWORDS_PICTURES_DIR", "/srv/pictures")

	cfg := FromEnv()

	if cfg.Addr != "0.0.0.0:9999" {
		t.Fatalf("expected env addr override, got %q", cfg.Addr)
	}
	if cfg.DatabasePath != "/tmp/codewords.sqlite" {
		t.Fatalf("expected env database path override, got %q", cfg.DatabasePath)
	}
	if cfg.PicturesDir != "/srv/pictures" {
		t.Fatalf("expected env pictures dir override, got %q", cfg.PicturesDir)
	}
}
