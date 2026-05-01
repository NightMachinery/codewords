# Assets and Wordpacks

## Wordpacks

Copy all existing SecretCodes wordpack `.txt` files from the FreeBoardGames repo into `assets/wordpacks/`.

Source path:

- `/home/ubuntu/base/FreeBoardGames.org/web/src/games/secretcodes/wordpacks/`

Requirements:

- Preserve non-empty words/phrases and comments where practical.
- Include non-English packs such as German, Dutch, Czech, Persian/Farsi, Harry Potter, adult/custom packs, and English variants.
- UI remains English only; wordpack names can be filenames or simple labels derived from filenames.
- Parse wordpacks by trimming each line, ignoring blank lines, and ignoring comment lines beginning with `#`.
- Prefer the mined legacy display order for bundled packs: `english`, `english-alternative`, `dutch`, `czech`, `german`, `persian-1`, `harry-potter-1`, `harry-potter-1-fa`, then any additional packs alphabetically.
- Do not fetch wordpacks remotely.

## Static frontend assets

- Bundle all app assets locally in the repo.
- Use locally bundled fonts or system fonts only.
- No external icon/font/CDN references.
- Remove donation links, previous creator names in UI, and propaganda content if copied accidentally.

## Picture mode

- Picture mode must use local files only.
- Support a configured local source directory and a local cache directory.
- Default candidate source can mirror the old convention, but must be configurable.
- Discover common local image types (`.jpg`, `.jpeg`, `.png`, `.webp`) and optionally sniff supported extensionless files.
- Normalize/cache images server-side if implemented; generated cache files live outside source assets or in a documented cache directory.
- Prefer the reverse-mined AVIF cache semantics unless there is a deliberate documented simplification: stable opaque ids from source bytes plus transform/version descriptor, central crop to 2:3, normalized 1024x1536 output, content type `image/avif`, and cache files under a configurable local cache directory.
- If cache-hit validation is implemented, make it explicit and testable; the legacy trigger was `FBG_VALIDATE_CACHE_HITS_P` and rebuilt bad cache entries only when enabled.
- Expose only safe image ids to clients, not arbitrary filesystem paths.
- Require at least as many unique catalog images as the requested `imageCardCount`; image-only mode requires 25.

## What else to copy from the old repo

The wordpack directory is the only required direct copy from the old SecretCodes implementation.

Do **not** directly copy the old React/boardgame.io implementation files, locale directories, tests, or server picture pipeline into the new app. They are useful only as behavioral reference because the new project uses a different stack and English-only UI.

Optional assets:

- `web/src/games/secretcodes/media/thumbnail.jpg` may be copied later if a thumbnail is needed and the image is acceptable for the new Codewords branding.
- `web/src/games/secretcodes/locales/en.json` may be used as wording reference, but the new UI should define its own English strings without localization infrastructure.

## Reverse-mined source reference

Detailed behavior mined from the original project is documented in `../docs/specs/secretcodes-reverse-spec.md`. Use that file as behavioral evidence, not as permission to copy the old React/boardgame.io implementation.

## Mixed card assets

Mixed mode uses the same bundled wordpacks and local picture catalog as the dedicated modes. No external image lookup or remote asset service is allowed. The server chooses the exact mix at match start and persists concrete card contents in the match snapshot.
