package server

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"mime"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const avifTransformDescriptorFormat = "source=%s|ratio=2:3|long_side=1536|output=1024x1536|fmt=avif|backend=native|quality=80|speed=6|threads=auto|channels=rgb|pipeline=v1"

type pictureCatalog struct {
	images map[string]pictureAsset
	ids    []string
}

type pictureAsset struct {
	ID          string
	SourcePath  string
	CachePath   string
	ContentType string
}

type pictureCatalogOptions struct {
	ImageDir      string
	ImageCacheDir string
	ProcessAVIF   bool
}

// GenerateAVIFCache checks and builds the local AVIF cache without starting the backend.
func GenerateAVIFCache(imageDir, cacheDir string) error {
	_, err := loadPictureCatalog(pictureCatalogOptions{ImageDir: imageDir, ImageCacheDir: cacheDir, ProcessAVIF: true})
	return err
}

func loadPictureCatalog(opts pictureCatalogOptions) (*pictureCatalog, error) {
	if strings.TrimSpace(opts.ImageDir) == "" || strings.TrimSpace(opts.ImageCacheDir) == "" {
		return &pictureCatalog{images: map[string]pictureAsset{}}, nil
	}
	entries, err := discoverPictureFiles(opts.ImageDir)
	if err != nil {
		return nil, err
	}
	if opts.ProcessAVIF {
		if err := os.MkdirAll(opts.ImageCacheDir, 0o755); err != nil {
			return nil, fmt.Errorf("create image cache dir: %w", err)
		}
	}
	catalog := &pictureCatalog{images: map[string]pictureAsset{}}
	for _, path := range entries {
		bytes, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read picture %s: %w", path, err)
		}
		id := legacyImageID(bytes)
		cachePath := filepath.Join(opts.ImageCacheDir, id+".avif")
		if opts.ProcessAVIF {
			if err := ensureAVIFCache(path, cachePath); err != nil {
				return nil, err
			}
		} else if _, err := os.Stat(cachePath); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("stat cached picture %s: %w", cachePath, err)
		}
		if _, exists := catalog.images[id]; exists {
			continue
		}
		catalog.images[id] = pictureAsset{ID: id, SourcePath: path, CachePath: cachePath, ContentType: "image/avif"}
		catalog.ids = append(catalog.ids, id)
	}
	sort.Strings(catalog.ids)
	return catalog, nil
}

func discoverPictureFiles(dir string) ([]string, error) {
	var out []string
	if err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if supportedImagePath(path) {
			out = append(out, path)
			return nil
		}
		if filepath.Ext(path) == "" {
			ok, err := sniffSupportedImage(path)
			if err != nil {
				return err
			}
			if ok {
				out = append(out, path)
			}
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("discover pictures: %w", err)
	}
	sort.Strings(out)
	return out, nil
}

func supportedImagePath(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp":
		return true
	default:
		ct := mime.TypeByExtension(ext)
		return strings.HasPrefix(ct, "image/") && ct != "image/avif"
	}
}

func sniffSupportedImage(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, fmt.Errorf("open image for sniffing %s: %w", path, err)
	}
	defer f.Close()
	buf := make([]byte, 16)
	n, err := f.Read(buf)
	if err != nil && n == 0 {
		return false, nil
	}
	buf = buf[:n]
	return bytes.HasPrefix(buf, []byte{0xff, 0xd8, 0xff}) || bytes.HasPrefix(buf, []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}) || (len(buf) >= 12 && string(buf[:4]) == "RIFF" && string(buf[8:12]) == "WEBP"), nil
}

func legacyImageID(sourceBytes []byte) string {
	sourceSum := sha256.Sum256(sourceBytes)
	descriptor := fmt.Sprintf(avifTransformDescriptorFormat, hex.EncodeToString(sourceSum[:]))
	idSum := sha256.Sum256([]byte(descriptor))
	return hex.EncodeToString(idSum[:])
}

func ensureAVIFCache(sourcePath, cachePath string) error {
	if ok, err := validCachedAVIF(cachePath); err == nil && ok {
		return nil
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := buildAVIFCache(sourcePath, cachePath); err != nil {
		return err
	}
	ok, err := validCachedAVIF(cachePath)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("built avif cache has wrong dimensions: %s", cachePath)
	}
	return nil
}

func validCachedAVIF(path string) (bool, error) {
	if _, err := os.Stat(path); err != nil {
		return false, err
	}
	identify, err := exec.LookPath("identify")
	if err != nil {
		return false, fmt.Errorf("check avif cache: missing identify command: %w", err)
	}
	out, err := exec.Command(identify, "-format", "%wx%h", path).CombinedOutput()
	if err != nil {
		return false, nil
	}
	return strings.TrimSpace(string(out)) == "1024x1536", nil
}

func buildAVIFCache(sourcePath, cachePath string) error {
	convert, err := exec.LookPath("convert")
	if err != nil {
		return fmt.Errorf("build avif cache: missing convert command: %w", err)
	}
	avifenc, err := exec.LookPath("avifenc")
	if err != nil {
		return fmt.Errorf("build avif cache: missing avifenc command: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
		return fmt.Errorf("create avif cache dir: %w", err)
	}
	tmpDir := filepath.Dir(cachePath)
	tmpPNG, err := os.CreateTemp(tmpDir, ".codewords-*.png")
	if err != nil {
		return fmt.Errorf("create temp png: %w", err)
	}
	tmpPNGPath := tmpPNG.Name()
	_ = tmpPNG.Close()
	defer os.Remove(tmpPNGPath)
	tmpAVIF, err := os.CreateTemp(tmpDir, ".codewords-*.avif")
	if err != nil {
		return fmt.Errorf("create temp avif: %w", err)
	}
	tmpAVIFPath := tmpAVIF.Name()
	_ = tmpAVIF.Close()
	defer os.Remove(tmpAVIFPath)
	if out, err := exec.Command(convert, sourcePath, "-auto-orient", "-resize", "1024x1536^", "-gravity", "center", "-extent", "1024x1536", tmpPNGPath).CombinedOutput(); err != nil {
		return fmt.Errorf("convert image to 2:3 png: %w: %s", err, strings.TrimSpace(string(out)))
	}
	if out, err := exec.Command(avifenc, "-q", "80", "--speed", "6", tmpPNGPath, tmpAVIFPath).CombinedOutput(); err != nil {
		return fmt.Errorf("encode avif cache: %w: %s", err, strings.TrimSpace(string(out)))
	}
	if err := os.Rename(tmpAVIFPath, cachePath); err != nil {
		return fmt.Errorf("install avif cache: %w", err)
	}
	return nil
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
