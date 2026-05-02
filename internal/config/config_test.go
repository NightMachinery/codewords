package config

import "testing"

func TestFromEnvUsesDefaults(t *testing.T) {
	t.Setenv("CODEWORDS_ADDR", "")
	t.Setenv("CODEWORDS_DATABASE_PATH", "")
	t.Setenv("CODEWORDS_IMAGE_DIR", "")
	t.Setenv("CODEWORDS_IMAGE_CACHE_DIR", "")
	t.Setenv("CODEWORDS_AVIF_PROCESS_P", "")

	cfg := FromEnv()

	if cfg.Addr != defaultAddr {
		t.Fatalf("expected default addr %q, got %q", defaultAddr, cfg.Addr)
	}
	if cfg.DatabasePath != defaultDatabasePath {
		t.Fatalf("expected default database path %q, got %q", defaultDatabasePath, cfg.DatabasePath)
	}
	if cfg.ImageDir != "" {
		t.Fatalf("expected picture mode disabled by default, got image dir %q", cfg.ImageDir)
	}
	if cfg.ImageCacheDir != "" {
		t.Fatalf("expected no backend cache dir default, got %q", cfg.ImageCacheDir)
	}
	if cfg.AVIFProcess {
		t.Fatalf("expected backend AVIF processing disabled by default")
	}
}

func TestFromEnvOverridesValues(t *testing.T) {
	t.Setenv("CODEWORDS_ADDR", "0.0.0.0:9999")
	t.Setenv("CODEWORDS_DATABASE_PATH", "/tmp/codewords.sqlite")
	t.Setenv("CODEWORDS_IMAGE_DIR", "/srv/pictures")
	t.Setenv("CODEWORDS_IMAGE_CACHE_DIR", "/srv/cache")
	t.Setenv("CODEWORDS_AVIF_PROCESS_P", "y")

	cfg := FromEnv()

	if cfg.Addr != "0.0.0.0:9999" {
		t.Fatalf("expected env addr override, got %q", cfg.Addr)
	}
	if cfg.DatabasePath != "/tmp/codewords.sqlite" {
		t.Fatalf("expected env database path override, got %q", cfg.DatabasePath)
	}
	if cfg.ImageDir != "/srv/pictures" {
		t.Fatalf("expected env pictures dir override, got %q", cfg.ImageDir)
	}
	if cfg.ImageCacheDir != "/srv/cache" {
		t.Fatalf("expected env cache dir override, got %q", cfg.ImageCacheDir)
	}
	if !cfg.AVIFProcess {
		t.Fatalf("expected AVIF processing override")
	}
}
