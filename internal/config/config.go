package config

import "os"

const (
	defaultAddr         = "127.0.0.1:7878"
	defaultDatabasePath = "./data/codewords.sqlite"
	defaultPicturesDir  = "./assets/pictures"
)

// Config contains runtime settings for the Codewords server.
type Config struct {
	Addr         string
	DatabasePath string
	PicturesDir  string
}

// FromEnv loads configuration from environment variables.
func FromEnv() Config {
	addr := os.Getenv("CODEWORDS_ADDR")
	if addr == "" {
		addr = defaultAddr
	}
	databasePath := os.Getenv("CODEWORDS_DATABASE_PATH")
	if databasePath == "" {
		databasePath = defaultDatabasePath
	}
	picturesDir := os.Getenv("CODEWORDS_PICTURES_DIR")
	if picturesDir == "" {
		picturesDir = defaultPicturesDir
	}
	return Config{Addr: addr, DatabasePath: databasePath, PicturesDir: picturesDir}
}
