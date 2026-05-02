package main

import (
	"context"
	"log"
	"net/http"

	"github.com/NightMachinery/codewords/internal/config"
	"github.com/NightMachinery/codewords/internal/identity"
	"github.com/NightMachinery/codewords/internal/server"
	"github.com/NightMachinery/codewords/internal/storage"
)

func main() {
	cfg := config.FromEnv()
	db, err := storage.Open(context.Background(), cfg.DatabasePath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	identityService := identity.NewService(db, identity.Options{})
	handler, err := server.NewHandler(server.Options{Store: db, Identity: identityService, WordpacksDir: "assets/wordpacks", PicturesDir: cfg.PicturesDir})
	if err != nil {
		log.Fatalf("configure server: %v", err)
	}

	log.Printf("codewords server listening on %s", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, handler); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
