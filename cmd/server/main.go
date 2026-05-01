package main

import (
	"log"
	"net/http"

	"github.com/NightMachinery/codewords/internal/config"
	"github.com/NightMachinery/codewords/internal/server"
)

func main() {
	cfg := config.FromEnv()
	handler := server.NewHandler()

	log.Printf("codewords server listening on %s", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, handler); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
