package config

import "os"

const defaultAddr = "127.0.0.1:7878"

// Config contains runtime settings for the Codewords server.
type Config struct {
	Addr string
}

// FromEnv loads configuration from environment variables.
func FromEnv() Config {
	addr := os.Getenv("CODEWORDS_ADDR")
	if addr == "" {
		addr = defaultAddr
	}
	return Config{Addr: addr}
}
