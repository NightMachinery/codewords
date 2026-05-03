package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/NightMachinery/codewords/internal/config"
	"github.com/NightMachinery/codewords/internal/identity"
	"github.com/NightMachinery/codewords/internal/server"
	"github.com/NightMachinery/codewords/internal/storage"
)

func main() {
	if len(os.Args) > 1 {
		runSubcommand(os.Args[1:])
		return
	}
	runServer()
}

func runServer() {
	cfg := config.FromEnv()
	db, err := storage.Open(context.Background(), cfg.DatabasePath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	identityService := identity.NewService(db, identity.Options{})
	handler, err := server.NewHandler(server.Options{Store: db, Identity: identityService, WordpacksDir: "assets/wordpacks", ImageDir: cfg.ImageDir, ImageCacheDir: cfg.ImageCacheDir, AVIFProcess: cfg.AVIFProcess, LogPictures: true})
	if err != nil {
		log.Fatalf("configure server: %v", err)
	}
	if line, ok := server.PictureDiagnostics(handler); ok {
		log.Print(line)
	}

	log.Printf("codewords server listening on %s", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, handler); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func runSubcommand(args []string) {
	if len(args) == 2 && args[0] == "avif-cache" && args[1] == "gen" {
		cfg := config.FromEnv()
		if cfg.ImageDir == "" || cfg.ImageCacheDir == "" {
			log.Fatalf("CODEWORDS_IMAGE_DIR and CODEWORDS_IMAGE_CACHE_DIR are required")
		}
		if err := server.GenerateAVIFCache(cfg.ImageDir, cfg.ImageCacheDir); err != nil {
			log.Fatalf("generate avif cache: %v", err)
		}
		log.Printf("AVIF cache ready for %s in %s", cfg.ImageDir, cfg.ImageCacheDir)
		return
	}
	fmt.Fprintf(os.Stderr, "Usage:\n  codewords\n  codewords avif-cache gen\n")
	os.Exit(2)
}
