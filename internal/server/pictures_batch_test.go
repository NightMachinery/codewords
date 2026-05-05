package server

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseIdentifyOutput(t *testing.T) {
	good := "00187a4b0245f9af4bbad7db487cd19a3531539b0623ca39a47f10617f978604.avif"
	other := "8b9f0e33a6d2e0dcb0b90e06dd5f53e0e0623cbd42fa4608e0ff96bb9c836b5e.avif"
	tests := []struct {
		name      string
		out       string
		requested []string
		wantErr   string
		want      map[string]dimensionResult
	}{
		{
			name:      "valid rows",
			out:       good + identifyFieldSep + "1024" + identifyFieldSep + "1536\n" + other + identifyFieldSep + "800" + identifyFieldSep + "600\n",
			requested: []string{good, other},
			want: map[string]dimensionResult{
				good:  {Width: 1024, Height: 1536},
				other: {Width: 800, Height: 600},
			},
		},
		{
			name:      "missing requested row is allowed by parser",
			out:       good + identifyFieldSep + "1024" + identifyFieldSep + "1536\n",
			requested: []string{good, other},
			want: map[string]dimensionResult{
				good: {Width: 1024, Height: 1536},
			},
		},
		{name: "malformed row", out: "broken\n", requested: []string{good}, wantErr: "malformed row"},
		{name: "duplicate row", out: good + identifyFieldSep + "1024" + identifyFieldSep + "1536\n" + good + identifyFieldSep + "1024" + identifyFieldSep + "1536\n", requested: []string{good}, wantErr: "duplicate identify row"},
		{name: "unexpected filename", out: other + identifyFieldSep + "1024" + identifyFieldSep + "1536\n", requested: []string{good}, wantErr: "unexpected filename"},
		{name: "bad width", out: good + identifyFieldSep + "wide" + identifyFieldSep + "1536\n", requested: []string{good}, wantErr: "bad width"},
		{name: "bad height", out: good + identifyFieldSep + "1024" + identifyFieldSep + "tall\n", requested: []string{good}, wantErr: "bad height"},
		{name: "non-positive dimensions", out: good + identifyFieldSep + "0" + identifyFieldSep + "1536\n", requested: []string{good}, wantErr: "non-positive dimensions"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseIdentifyOutput(tt.out, tt.requested)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("parse identify output: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %#v want %#v", got, tt.want)
			}
			for name, want := range tt.want {
				if got[name].Width != want.Width || got[name].Height != want.Height || got[name].Err != nil {
					t.Fatalf("result for %s = %#v, want %#v", name, got[name], want)
				}
			}
		})
	}
}

func TestRunIdentifyBatchUsesFilenameTaggedRowsWithNonZeroExit(t *testing.T) {
	cacheDir := t.TempDir()
	good := "00187a4b0245f9af4bbad7db487cd19a3531539b0623ca39a47f10617f978604.avif"
	bad := "8b9f0e33a6d2e0dcb0b90e06dd5f53e0e0623cbd42fa4608e0ff96bb9c836b5e.avif"
	script := filepath.Join(t.TempDir(), "identify-fake")
	body := "#!/bin/sh\nprintf '%s%s1024%s1536\\n' '" + good + "' '" + identifyFieldSep + "' '" + identifyFieldSep + "'\necho 'bad file' >&2\nexit 1\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatalf("write fake identify: %v", err)
	}

	results := runIdentifyBatch(context.Background(), script, cacheDir, []string{good, bad})

	if !isValidAVIFResult(results[good]) {
		t.Fatalf("expected valid row to be used despite non-zero exit, got %#v", results[good])
	}
	if results[bad].Err == nil || !strings.Contains(results[bad].Err.Error(), "produced no row") {
		t.Fatalf("expected missing row to be invalid, got %#v", results[bad])
	}
}

func TestCheckExpectedAVIFCachesRejectsInvalidBasenameBeforeBatching(t *testing.T) {
	cacheDir := t.TempDir()
	checker := &recordingDimensionChecker{results: map[string]dimensionResult{}}
	cachePath := filepath.Join(cacheDir, "not-a-cache-name.avif")
	if err := os.WriteFile(cachePath, []byte("avif"), 0o644); err != nil {
		t.Fatalf("write cache fixture: %v", err)
	}

	results, err := checkExpectedAVIFCaches(context.Background(), checker, []expectedPictureCache{{CachePath: cachePath}}, pictureCatalogOptions{})
	if err != nil {
		t.Fatalf("check caches: %v", err)
	}

	if checker.calls != 0 {
		t.Fatalf("expected invalid basename to be rejected before batching, got %d calls", checker.calls)
	}
	if results[cachePath].Err == nil || !strings.Contains(results[cachePath].Err.Error(), "invalid cache basename") {
		t.Fatalf("expected invalid basename result, got %#v", results[cachePath])
	}
}

func TestProcessingDisabledDoesNotCallDimensionChecker(t *testing.T) {
	imageDir := t.TempDir()
	cacheDir := t.TempDir()
	sourceBytes := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0, 0, 0, 0}
	if err := os.WriteFile(filepath.Join(imageDir, "card.png"), sourceBytes, 0o644); err != nil {
		t.Fatalf("write source image: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, legacyImageID(sourceBytes)+".avif"), []byte("cached avif"), 0o644); err != nil {
		t.Fatalf("write cache fixture: %v", err)
	}
	checker := &recordingDimensionChecker{err: errors.New("checker should not be called")}

	catalog, err := loadPictureCatalog(pictureCatalogOptions{ImageDir: imageDir, ImageCacheDir: cacheDir, DimensionChecker: checker})
	if err != nil {
		t.Fatalf("load picture catalog: %v", err)
	}

	if checker.calls != 0 {
		t.Fatalf("expected processing-disabled path not to call checker, got %d calls", checker.calls)
	}
	if len(catalog.ids) != 1 {
		t.Fatalf("expected cached image to be exposed, got %#v", catalog.ids)
	}
}

func TestProcessingDisabledDefersCacheExistenceChecks(t *testing.T) {
	imageDir := t.TempDir()
	cacheFile := filepath.Join(t.TempDir(), "cache-is-file")
	sourceBytes := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0, 0, 0, 0}
	if err := os.WriteFile(filepath.Join(imageDir, "card.png"), sourceBytes, 0o644); err != nil {
		t.Fatalf("write source image: %v", err)
	}
	if err := os.WriteFile(cacheFile, []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("write cache file fixture: %v", err)
	}
	checker := &recordingDimensionChecker{err: errors.New("checker should not be called")}

	catalog, err := loadPictureCatalog(pictureCatalogOptions{ImageDir: imageDir, ImageCacheDir: cacheFile, ProcessAVIF: false, DimensionChecker: checker})
	if err != nil {
		t.Fatalf("load picture catalog should not stat cache paths while processing is disabled: %v", err)
	}

	if checker.calls != 0 {
		t.Fatalf("expected processing-disabled path not to call checker, got %d calls", checker.calls)
	}
	if len(catalog.ids) != 1 || catalog.ids[0] != legacyImageID(sourceBytes) {
		t.Fatalf("expected uncached source candidate to be exposed, got %#v", catalog.ids)
	}
	if catalog.Diagnostics().CacheHitCount != 0 || catalog.Diagnostics().CacheMissCount != 0 {
		t.Fatalf("expected no startup cache hit/miss checks, got %#v", catalog.Diagnostics())
	}
}

func TestProcessingEnabledUsesBatchedDimensionCheckerForValidCaches(t *testing.T) {
	imageDir := t.TempDir()
	cacheDir := t.TempDir()
	sourceOne := []byte("source-one")
	sourceTwo := []byte("source-two")
	if err := os.WriteFile(filepath.Join(imageDir, "one.png"), sourceOne, 0o644); err != nil {
		t.Fatalf("write source one: %v", err)
	}
	if err := os.WriteFile(filepath.Join(imageDir, "two.png"), sourceTwo, 0o644); err != nil {
		t.Fatalf("write source two: %v", err)
	}
	cacheOne := legacyImageID(sourceOne) + ".avif"
	cacheTwo := legacyImageID(sourceTwo) + ".avif"
	if err := os.WriteFile(filepath.Join(cacheDir, cacheOne), []byte("cached one"), 0o644); err != nil {
		t.Fatalf("write cache one: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, cacheTwo), []byte("cached two"), 0o644); err != nil {
		t.Fatalf("write cache two: %v", err)
	}
	checker := &recordingDimensionChecker{results: map[string]dimensionResult{
		cacheOne: {Width: expectedAVIFWidth, Height: expectedAVIFHeight},
		cacheTwo: {Width: expectedAVIFWidth, Height: expectedAVIFHeight},
	}}

	catalog, err := loadPictureCatalog(pictureCatalogOptions{ImageDir: imageDir, ImageCacheDir: cacheDir, ProcessAVIF: true, DimensionChecker: checker})
	if err != nil {
		t.Fatalf("load picture catalog: %v", err)
	}

	if checker.calls != 1 {
		t.Fatalf("expected one batched checker call, got %d", checker.calls)
	}
	if len(checker.batches) != 1 || len(checker.batches[0]) != 2 {
		t.Fatalf("expected both cache files in one batch, got %#v", checker.batches)
	}
	if len(catalog.ids) != 2 {
		t.Fatalf("expected both valid cached images to be exposed, got %#v", catalog.ids)
	}
	if catalog.Diagnostics().CacheHitCount != 2 || catalog.Diagnostics().CacheMissCount != 0 {
		t.Fatalf("expected two cache hits and no misses, got %#v", catalog.Diagnostics())
	}
}

type recordingDimensionChecker struct {
	calls   int
	batches [][]string
	results map[string]dimensionResult
	err     error
}

func (c *recordingDimensionChecker) CheckBatch(ctx context.Context, cacheDir string, basenames []string, opts pictureCatalogOptions) map[string]dimensionResult {
	c.calls++
	c.batches = append(c.batches, append([]string(nil), basenames...))
	out := make(map[string]dimensionResult, len(basenames))
	for _, basename := range basenames {
		if c.err != nil {
			out[basename] = dimensionResult{Err: c.err}
			continue
		}
		if res, ok := c.results[basename]; ok {
			out[basename] = res
			continue
		}
		out[basename] = dimensionResult{Err: errors.New("missing test result")}
	}
	return out
}
