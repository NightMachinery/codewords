package config

import "testing"

func TestFromEnvUsesDefaultAddress(t *testing.T) {
	t.Setenv("CODEWORDS_ADDR", "")

	cfg := FromEnv()

	if cfg.Addr != "127.0.0.1:7878" {
		t.Fatalf("expected default addr 127.0.0.1:7878, got %q", cfg.Addr)
	}
}

func TestFromEnvUsesConfiguredAddress(t *testing.T) {
	t.Setenv("CODEWORDS_ADDR", "0.0.0.0:9000")

	cfg := FromEnv()

	if cfg.Addr != "0.0.0.0:9000" {
		t.Fatalf("expected configured addr, got %q", cfg.Addr)
	}
}
