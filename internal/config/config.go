package config

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultAddr         = "127.0.0.1:7878"
	defaultDatabasePath = "./data/codewords.sqlite"
)

// Config contains runtime settings for the Codewords server.
type Config struct {
	Addr          string
	DatabasePath  string
	ImageDir      string
	ImageCacheDir string
	AVIFProcess   bool
}

// FromEnv loads configuration from environment variables.
func FromEnv() Config {
	addr := os.Getenv("CODEWORDS_ADDR")
	if addr == "" {
		addr = defaultAddr
	}
	databasePath := os.Getenv("CODEWORDS_DATABASE_PATH")
	if databasePath == "" {
		if dataDir := os.Getenv("CODEWORDS_DATA_DIR"); dataDir != "" {
			databasePath = filepath.Join(expandHome(dataDir), "codewords.sqlite")
		} else {
			databasePath = defaultDatabasePath
		}
	}
	return Config{
		Addr:          addr,
		DatabasePath:  expandHome(databasePath),
		ImageDir:      expandHome(os.Getenv("CODEWORDS_IMAGE_DIR")),
		ImageCacheDir: expandHome(os.Getenv("CODEWORDS_IMAGE_CACHE_DIR")),
		AVIFProcess:   parseTruthy(os.Getenv("CODEWORDS_AVIF_PROCESS_P")),
	}
}

func parseTruthy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "y", "yes", "true", "1":
		return true
	default:
		return false
	}
}

func expandHome(path string) string {
	if path == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
	}
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
	}
	return path
}
