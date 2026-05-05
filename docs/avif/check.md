# AVIF Cache Checking

Codewords serves local picture cards only from cached AVIF files. With AVIF processing enabled, cache validation happens while building the in-memory picture catalog. With AVIF processing disabled, startup only discovers source candidates and defers image id hashing plus cache existence checks until a game starts.

## When checks run

- `CODEWORDS_AVIF_PROCESS_P=y` on server startup: discover source images, verify each matching AVIF cache file, and rebuild missing or invalid cache files.
- `bin/codewords avif-cache gen`: same checking and rebuilding path as startup with processing enabled.
- `CODEWORDS_AVIF_PROCESS_P` unset/false on server startup: discover source images but defer cache existence checks until a game starts; do not validate dimensions or rebuild files.

### What Happens if I Set `Export CODEWORDS_AVIF_PROCESS_P=n`? Are Already Cached  Images Still Loaded, Just Without Validation?
  Yes — with export CODEWORDS_AVIF_PROCESS_P=n, already cached images are still
  loaded, but only if their matching source image is discoverable.

  Behavior with n:

  - The backend still scans CODEWORDS_IMAGE_DIR recursively and follows symlinked
    directories.
  - It does not read every source image or compute every legacy imageId at startup.
  - When a game starts with image cards, source candidates are shuffled from the game seed. For selected candidates and replacements, the backend reads the source bytes, computes the legacy imageId, and computes the matching cache path:

  <CODEWORDS_IMAGE_CACHE_DIR>/<imageId>.avif

  - The catalog endpoint reports that source candidates are available, but disabled mode does not expose per-image ids before start.
  - Selected candidates are checked in bounded parallel batches, and missing cache files are skipped in favor of later shuffled replacements.
  - If not enough cached candidates exist, image/mixed game start fails.
  - It does not run identify.
  - It does not validate dimensions.
  - It does not run convert.
  - It does not run avifenc.
  - It does not rebuild missing/corrupt/wrong-size cache files.

  So CODEWORDS_AVIF_PROCESS_P=n is the fast normal startup mode after your cache has
  already been generated.

  One important caveat: orphan AVIFs in the cache dir are not loaded by themselves.
  The original source images must still be reachable from CODEWORDS_IMAGE_DIR,
  because the cache filename is derived from source bytes.

## Discovery step

The backend scans `CODEWORDS_IMAGE_DIR` recursively and follows symlinked directories. It accepts:

- `.jpg`
- `.jpeg`
- `.png`
- `.webp`
- other file extensions whose MIME type is `image/*`, except `.avif`
- extensionless files that sniff as JPEG, PNG, or WebP

When AVIF processing is enabled, every source image is read during startup so the backend can compute the legacy-compatible image id. When processing is disabled, this read/hash step is deferred until match start for selected candidates and replacements. The cache path is then:

```text
<CODEWORDS_IMAGE_CACHE_DIR>/<imageId>.avif
```

The id is based on the source bytes plus the fixed transform descriptor, so an AVIF file in the cache directory cannot be matched unless the original source image is discoverable.

## Fast path when processing is off

When `CODEWORDS_AVIF_PROCESS_P` is false, startup does not call `os.Stat` for every expected cache path.

- Discoverable source image: counted by the picture catalog as a candidate.
- Existing `<imageId>.avif`: eligible to be selected at match start.
- Missing `<imageId>.avif`: skipped at match start and replaced with a later shuffled candidate.
- No ImageMagick or `avifenc` commands are run.
- Existing cache bytes are trusted; dimensions are not checked.

This is the intended normal startup mode after a cache has already been generated. It keeps startup fast for large catalogs while preserving per-game shuffled image selection.

## Full check when processing is on

When `CODEWORDS_AVIF_PROCESS_P` is true, expected cache files are pre-statted first. Missing files are marked for rebuild. Existing files are validated in batches with ImageMagick `identify` from the cache directory, passing only generated cache basenames.

The batch command emits filename-tagged width and height records:

```sh
identify -format '%f|||CODEWORDS_AVIF_DIM|||%w|||CODEWORDS_AVIF_DIM|||%h\n' -- \
  00187a4b0245f9af4bbad7db487cd19a3531539b0623ca39a47f10617f978604.avif \
  8b9f0e33a6d2e0dcb0b90e06dd5f53e0e0623cbd42fa4608e0ff96bb9c836b5e.avif
```

The parser keys results by filename, not row position. This matters because `identify` can return a non-zero exit and still print valid rows for other inputs, or omit rows for corrupt/unreadable inputs.

Validation rules:

- Missing cache file after `os.Stat`: rebuild.
- Existing cache file with basename outside `^[0-9a-f]{64}\.avif$`: rebuild.
- Filename-tagged row with `1024x1536`: keep.
- Filename-tagged row with any other dimensions: rebuild.
- Requested basename missing from stdout: rebuild.
- Malformed, duplicate, or unexpected output rows: fail closed for the whole chunk.

Files are checked in chunks of 128 basenames. After rebuilding a missing or invalid file, Codewords still performs a single-file dimension check for that rebuilt file.

## Rebuild path

A rebuild uses ImageMagick `convert` and `avifenc`:

```sh
convert <sourcePath> -auto-orient -resize '1024x1536^' -gravity center -extent 1024x1536 <tmp.png>
avifenc -q 80 --speed 6 <tmp.png> <tmp.avif>
```

The temporary AVIF is then renamed into the final cache path.

The normalized output contract is:

- aspect ratio: 2:3
- output size: `1024x1536`
- format: AVIF
- quality: `80`
- speed: `6`

## Why it can be slow

The expensive parts are:

- reading source images to compute legacy image ids (all images when processing is enabled; only selected candidates/replacements when disabled)
- reading/parsing every existing AVIF cache file when processing is enabled
- invoking external `identify` processes for batches of cache files, rather than one process per file
- rebuilding missing or invalid files with `convert` and `avifenc`

For a large catalog, prefer this workflow:

1. Run `bin/codewords avif-cache gen` during maintenance.
2. Start/redeploy the server with `CODEWORDS_AVIF_PROCESS_P` false so startup only discovers source candidates and defers cache existence checks to match start.

## Relevant code

- `internal/server/pictures.go`: catalog loading, source discovery, cache validation, and rebuild logic.
- `cmd/server/main.go`: server startup and manual `avif-cache gen` command.
- `internal/config/config.go`: `CODEWORDS_AVIF_PROCESS_P` parsing.
