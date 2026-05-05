package server

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"mime"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
)

const avifTransformDescriptorFormat = "source=%s|ratio=2:3|long_side=1536|output=1024x1536|fmt=avif|backend=native|quality=80|speed=6|threads=auto|channels=rgb|pipeline=v1"
const identifyFieldSep = "|||CODEWORDS_AVIF_DIM|||"
const identifyBatchSize = 128
const expectedAVIFWidth = 1024
const expectedAVIFHeight = 1536

var validAVIFCacheBasename = regexp.MustCompile(`^[0-9a-f]{64}\.avif$`)

type pictureCatalog struct {
	images map[string]pictureAsset
	ids    []string
	diag   pictureCatalogDiagnostics
}

type pictureAsset struct {
	ID          string
	SourcePath  string
	CachePath   string
	ContentType string
}

type pictureCatalogOptions struct {
	ImageDir         string
	ImageCacheDir    string
	ProcessAVIF      bool
	Logf             func(string, ...any)
	DimensionChecker avifDimensionBatchChecker
}

type pictureCatalogDiagnostics struct {
	ImageDir        string
	ImageCacheDir   string
	ProcessAVIF     bool
	SourceCount     int
	CachedAVIFCount int
	EnabledCount    int
	CacheHitCount   int
	CacheMissCount  int
	DuplicateCount  int
	DisabledReason  string
}

type avifDimensionBatchChecker interface {
	CheckBatch(ctx context.Context, cacheDir string, basenames []string, opts pictureCatalogOptions) map[string]dimensionResult
}

type identifyDimensionChecker struct {
	IdentifyBin string
	BatchSize   int
}

type dimensionResult struct {
	Width  int
	Height int
	Err    error
}

type expectedPictureCache struct {
	SourcePath string
	CachePath  string
}

// GenerateAVIFCache checks and builds the local AVIF cache without starting the backend.
func GenerateAVIFCache(imageDir, cacheDir string) error {
	_, err := loadPictureCatalog(pictureCatalogOptions{ImageDir: imageDir, ImageCacheDir: cacheDir, ProcessAVIF: true})
	return err
}

func loadPictureCatalog(opts pictureCatalogOptions) (*pictureCatalog, error) {
	diag := pictureCatalogDiagnostics{ImageDir: opts.ImageDir, ImageCacheDir: opts.ImageCacheDir, ProcessAVIF: opts.ProcessAVIF, CachedAVIFCount: countCachedAVIFFiles(opts.ImageCacheDir)}
	logPictureCatalog(opts, "image catalog: starting load image_dir=%q image_cache_dir=%q avif_processing=%t cached_avif_images=%d", opts.ImageDir, opts.ImageCacheDir, opts.ProcessAVIF, diag.CachedAVIFCount)
	if strings.TrimSpace(opts.ImageDir) == "" || strings.TrimSpace(opts.ImageCacheDir) == "" {
		diag.DisabledReason = catalogDisabledReason(diag)
		logPictureCatalog(opts, "image catalog: disabled before scan: %s", diag.DisabledReason)
		return &pictureCatalog{images: map[string]pictureAsset{}, diag: diag}, nil
	}
	logPictureCatalog(opts, "image catalog: scanning source images recursively and following symlinked directories")
	entries, err := discoverPictureFiles(opts.ImageDir)
	if err != nil {
		return nil, err
	}
	diag.SourceCount = len(entries)
	logPictureCatalog(opts, "image catalog: discovered %d supported source image(s)", diag.SourceCount)
	if opts.ProcessAVIF {
		if err := os.MkdirAll(opts.ImageCacheDir, 0o755); err != nil {
			return nil, fmt.Errorf("create image cache dir: %w", err)
		}
		logPictureCatalog(opts, "image catalog: AVIF processing is enabled; missing or invalid cache files will be generated")
	} else {
		logPictureCatalog(opts, "image catalog: AVIF processing is disabled; cache existence checks are deferred until match start")
	}
	catalog := &pictureCatalog{images: map[string]pictureAsset{}, diag: diag}
	expectedCaches := make([]expectedPictureCache, 0, len(entries))
	sourceByCachePath := make(map[string]string, len(entries))

	var mu sync.Mutex
	var firstErr error
	var wg sync.WaitGroup
	var progress atomic.Uint32

	sem := make(chan struct{}, 8) // Limit concurrency

	for _, path := range entries {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			mu.Lock()
			errVal := firstErr
			mu.Unlock()
			if errVal != nil {
				return
			}

			idx := progress.Add(1) - 1
			if shouldLogPictureProgress(int(idx), len(entries)) {
				logPictureCatalog(opts, "image catalog: checking source image %d/%d", idx+1, len(entries))
			}

			bytes, err := os.ReadFile(p)
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("read picture %s: %w", p, err)
				}
				mu.Unlock()
				return
			}
			id := legacyImageID(bytes)
			cachePath := filepath.Join(opts.ImageCacheDir, id+".avif")

			mu.Lock()
			defer mu.Unlock()
			if opts.ProcessAVIF {
				expectedCaches = append(expectedCaches, expectedPictureCache{SourcePath: p, CachePath: cachePath})
				sourceByCachePath[cachePath] = p
			} else {
				if _, exists := catalog.images[id]; exists {
					catalog.diag.DuplicateCount++
				} else {
					catalog.images[id] = pictureAsset{ID: id, SourcePath: p, CachePath: cachePath, ContentType: "image/avif"}
					catalog.ids = append(catalog.ids, id)
				}
			}
		}(path)
	}

	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}

	if opts.ProcessAVIF {
		results, err := checkExpectedAVIFCaches(context.Background(), dimensionCheckerOrDefault(opts.DimensionChecker), expectedCaches, opts)
		if err != nil {
			return nil, err
		}
		for _, expected := range expectedCaches {
			if isValidAVIFResult(results[expected.CachePath]) {
				catalog.diag.CacheHitCount++
				continue
			}
			catalog.diag.CacheMissCount++
			if err := buildAVIFCache(expected.SourcePath, expected.CachePath); err != nil {
				return nil, err
			}
			ok, err := validCachedAVIF(expected.CachePath)
			if err != nil {
				return nil, err
			}
			if !ok {
				return nil, fmt.Errorf("built avif cache has wrong dimensions: %s", expected.CachePath)
			}
		}
		catalog.images = map[string]pictureAsset{}
		catalog.ids = catalog.ids[:0]
		for _, expected := range expectedCaches {
			id := strings.TrimSuffix(filepath.Base(expected.CachePath), ".avif")
			if _, exists := catalog.images[id]; exists {
				catalog.diag.DuplicateCount++
				continue
			}
			catalog.images[id] = pictureAsset{ID: id, SourcePath: sourceByCachePath[expected.CachePath], CachePath: expected.CachePath, ContentType: "image/avif"}
			catalog.ids = append(catalog.ids, id)
		}
	}
	sort.Strings(catalog.ids)
	catalog.diag.EnabledCount = len(catalog.ids)
	catalog.diag.CachedAVIFCount = countCachedAVIFFiles(opts.ImageCacheDir)
	catalog.diag.DisabledReason = catalogDisabledReason(catalog.diag)
	logPictureCatalog(opts, "image catalog: finished load source_images=%d enabled_images=%d cache_hits=%d cache_misses=%d duplicates=%d cached_avif_images=%d", catalog.diag.SourceCount, catalog.diag.EnabledCount, catalog.diag.CacheHitCount, catalog.diag.CacheMissCount, catalog.diag.DuplicateCount, catalog.diag.CachedAVIFCount)
	if catalog.diag.DisabledReason != "" {
		logPictureCatalog(opts, "image catalog: image mode disabled because %s", catalog.diag.DisabledReason)
	} else {
		logPictureCatalog(opts, "image catalog: image mode enabled with %d image(s)", catalog.diag.EnabledCount)
	}
	return catalog, nil
}

func (c *pictureCatalog) Diagnostics() pictureCatalogDiagnostics {
	if c == nil {
		return pictureCatalogDiagnostics{DisabledReason: "picture catalog is not initialized"}
	}
	return c.diag
}

func (d pictureCatalogDiagnostics) StartupLogLine() string {
	state := "enabled"
	if d.EnabledCount == 0 {
		state = "disabled"
	}
	reason := d.DisabledReason
	if reason == "" {
		reason = "available"
	}
	return fmt.Sprintf("image mode %s: %s; source_images=%d enabled_images=%d cache_hits=%d cache_misses=%d duplicates=%d cached_avif_images=%d image_dir=%q image_cache_dir=%q avif_processing=%t", state, reason, d.SourceCount, d.EnabledCount, d.CacheHitCount, d.CacheMissCount, d.DuplicateCount, d.CachedAVIFCount, d.ImageDir, d.ImageCacheDir, d.ProcessAVIF)
}

func catalogDisabledReason(diag pictureCatalogDiagnostics) string {
	if diag.EnabledCount > 0 {
		return ""
	}
	if strings.TrimSpace(diag.ImageDir) == "" || strings.TrimSpace(diag.ImageCacheDir) == "" {
		return "CODEWORDS_IMAGE_DIR and CODEWORDS_IMAGE_CACHE_DIR must both be set"
	}
	if diag.SourceCount == 0 {
		return "no supported source images found in CODEWORDS_IMAGE_DIR; cached AVIF files alone cannot be matched without source images"
	}
	if !diag.ProcessAVIF {
		return "no cached AVIF files matched discovered source images and CODEWORDS_AVIF_PROCESS_P is not enabled"
	}
	return "no source images could be added to the catalog"
}

func countCachedAVIFFiles(dir string) int {
	if strings.TrimSpace(dir) == "" {
		return 0
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	count := 0
	for _, entry := range entries {
		if !entry.IsDir() && strings.EqualFold(filepath.Ext(entry.Name()), ".avif") {
			count++
		}
	}
	return count
}

func logPictureCatalog(opts pictureCatalogOptions, format string, args ...any) {
	if opts.Logf != nil {
		opts.Logf(format, args...)
	}
}

func shouldLogPictureProgress(index, total int) bool {
	if total == 0 {
		return false
	}
	if index == 0 || index == total-1 {
		return true
	}
	return (index+1)%100 == 0
}

func dimensionCheckerOrDefault(checker avifDimensionBatchChecker) avifDimensionBatchChecker {
	if checker != nil {
		return checker
	}
	return identifyDimensionChecker{IdentifyBin: "identify", BatchSize: identifyBatchSize}
}

func checkExpectedAVIFCaches(ctx context.Context, checker avifDimensionBatchChecker, expected []expectedPictureCache, opts pictureCatalogOptions) (map[string]dimensionResult, error) {
	results := make(map[string]dimensionResult, len(expected))
	byDir := map[string][]string{}
	pathByDirAndBase := map[string]map[string]string{}
	for _, cache := range expected {
		if _, err := os.Stat(cache.CachePath); err != nil {
			if os.IsNotExist(err) {
				results[cache.CachePath] = dimensionResult{Err: err}
				continue
			}
			return nil, fmt.Errorf("stat cached picture %s: %w", cache.CachePath, err)
		}
		dir := filepath.Dir(cache.CachePath)
		basename := filepath.Base(cache.CachePath)
		if !validAVIFCacheBasename.MatchString(basename) {
			results[cache.CachePath] = dimensionResult{Err: fmt.Errorf("invalid cache basename %q", basename)}
			continue
		}
		byDir[dir] = append(byDir[dir], basename)
		if pathByDirAndBase[dir] == nil {
			pathByDirAndBase[dir] = map[string]string{}
		}
		pathByDirAndBase[dir][basename] = cache.CachePath
	}
	var checkedCount int
	for dir, basenames := range byDir {
		batchResults := checker.CheckBatch(ctx, dir, basenames, opts)
		for _, basename := range basenames {
			path := pathByDirAndBase[dir][basename]
			res, ok := batchResults[basename]
			if !ok {
				res = dimensionResult{Err: fmt.Errorf("dimension checker produced no result for %q", basename)}
			}
			results[path] = res
		}
		checkedCount += len(basenames)
	}
	return results, nil
}

func (c identifyDimensionChecker) CheckBatch(ctx context.Context, cacheDir string, basenames []string, opts pictureCatalogOptions) map[string]dimensionResult {
	batchSize := c.BatchSize
	if batchSize <= 0 {
		batchSize = identifyBatchSize
	}
	identifyBin := c.IdentifyBin
	if identifyBin == "" {
		identifyBin = "identify"
	}
	results := make(map[string]dimensionResult, len(basenames))
	for start := 0; start < len(basenames); start += batchSize {
		end := start + batchSize
		if end > len(basenames) {
			end = len(basenames)
		}
		chunkResults := runIdentifyBatch(ctx, identifyBin, cacheDir, basenames[start:end])
		for name, res := range chunkResults {
			results[name] = res
		}
		logPictureCatalog(opts, "image catalog: verified cache file dimensions %d/%d", end, len(basenames))
	}
	return results
}

func runIdentifyBatch(ctx context.Context, identifyBin, cacheDir string, basenames []string) map[string]dimensionResult {
	format := "%f" + identifyFieldSep + "%w" + identifyFieldSep + "%h\n"
	args := []string{"-format", format, "--"}
	args = append(args, basenames...)
	cmd := exec.CommandContext(ctx, identifyBin, args...)
	cmd.Dir = cacheDir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()
	parsed, parseErr := parseIdentifyOutput(stdout.String(), basenames)
	if parseErr != nil {
		err := fmt.Errorf("parse identify output: %w; stderr: %s", parseErr, strings.TrimSpace(stderr.String()))
		return allInvalid(basenames, err)
	}
	results := make(map[string]dimensionResult, len(basenames))
	for _, name := range basenames {
		if res, ok := parsed[name]; ok {
			results[name] = res
			continue
		}
		err := fmt.Errorf("identify produced no row for %q", name)
		if runErr != nil {
			err = fmt.Errorf("%w; identify error: %v; stderr: %s", err, runErr, strings.TrimSpace(stderr.String()))
		}
		results[name] = dimensionResult{Err: err}
	}
	return results
}

func parseIdentifyOutput(out string, requested []string) (map[string]dimensionResult, error) {
	requestedSet := make(map[string]struct{}, len(requested))
	for _, name := range requested {
		requestedSet[name] = struct{}{}
	}
	results := make(map[string]dimensionResult)
	out = strings.TrimSuffix(out, "\n")
	if out == "" {
		return results, nil
	}
	for _, line := range strings.Split(out, "\n") {
		parts := strings.Split(line, identifyFieldSep)
		if len(parts) != 3 {
			return nil, fmt.Errorf("malformed row %q", line)
		}
		name := parts[0]
		if _, ok := requestedSet[name]; !ok {
			return nil, fmt.Errorf("unexpected filename in identify output %q", name)
		}
		if _, exists := results[name]; exists {
			return nil, fmt.Errorf("duplicate identify row for %q", name)
		}
		width, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("bad width for %q: %q", name, parts[1])
		}
		height, err := strconv.Atoi(parts[2])
		if err != nil {
			return nil, fmt.Errorf("bad height for %q: %q", name, parts[2])
		}
		if width <= 0 || height <= 0 {
			return nil, fmt.Errorf("non-positive dimensions for %q: %dx%d", name, width, height)
		}
		results[name] = dimensionResult{Width: width, Height: height}
	}
	return results, nil
}

func allInvalid(basenames []string, err error) map[string]dimensionResult {
	results := make(map[string]dimensionResult, len(basenames))
	for _, name := range basenames {
		results[name] = dimensionResult{Err: err}
	}
	return results
}

func isValidAVIFResult(res dimensionResult) bool {
	return res.Err == nil && res.Width == expectedAVIFWidth && res.Height == expectedAVIFHeight
}

func discoverPictureFiles(dir string) ([]string, error) {
	var out []string
	visited := map[fileIdentity]bool{}
	if err := walkPictureFiles(dir, visited, func(path string) error {
		if supportedImagePath(path) {
			out = append(out, path)
			return nil
		}
		if filepath.Ext(path) != "" {
			return nil
		}
		ok, err := sniffSupportedImage(path)
		if err != nil {
			return err
		}
		if ok {
			out = append(out, path)
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("discover pictures: %w", err)
	}
	sort.Strings(out)
	return out, nil
}

type fileIdentity struct {
	dev uint64
	ino uint64
}

func walkPictureFiles(path string, visited map[fileIdentity]bool, visitFile func(string) error) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return visitFile(path)
	}
	identity, ok := identityForFile(info)
	if ok {
		if visited[identity] {
			return nil
		}
		visited[identity] = true
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := walkPictureFiles(filepath.Join(path, entry.Name()), visited, visitFile); err != nil {
			return err
		}
	}
	return nil
}

func identityForFile(info os.FileInfo) (fileIdentity, bool) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fileIdentity{}, false
	}
	return fileIdentity{dev: uint64(stat.Dev), ino: uint64(stat.Ino)}, true
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
