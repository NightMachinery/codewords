package server

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type pictureCatalog struct {
	images map[string]pictureAsset
	ids    []string
}

type pictureAsset struct {
	ID          string
	Path        string
	ContentType string
}

func loadPictureCatalog(dir string) (*pictureCatalog, error) {
	if strings.TrimSpace(dir) == "" {
		return &pictureCatalog{images: map[string]pictureAsset{}}, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read picture dir: %w", err)
	}
	catalog := &pictureCatalog{images: map[string]pictureAsset{}}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		contentType := mime.TypeByExtension(ext)
		switch ext {
		case ".jpg", ".jpeg":
			contentType = "image/jpeg"
		case ".png":
			contentType = "image/png"
		case ".webp":
			contentType = "image/webp"
		default:
			if contentType == "" || !strings.HasPrefix(contentType, "image/") {
				continue
			}
		}
		path := filepath.Join(dir, entry.Name())
		bytes, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read picture %s: %w", path, err)
		}
		sum := sha256.Sum256(append([]byte(entry.Name()+"\x00"), bytes...))
		id := hex.EncodeToString(sum[:])
		catalog.images[id] = pictureAsset{ID: id, Path: path, ContentType: contentType}
		catalog.ids = append(catalog.ids, id)
	}
	sort.Strings(catalog.ids)
	return catalog, nil
}

func (c *pictureCatalog) listDTO(r *http.Request) []map[string]any {
	if c == nil {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(c.ids))
	for _, id := range c.ids {
		out = append(out, map[string]any{"id": id, "url": "/api/pictures/" + id})
	}
	return out
}
