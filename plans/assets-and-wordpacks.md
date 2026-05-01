# Assets and Wordpacks

## Wordpacks

Copy all existing SecretCodes wordpack `.txt` files from the FreeBoardGames repo into `assets/wordpacks/`.

Source path:

- `/home/ubuntu/base/FreeBoardGames.org/web/src/games/secretcodes/wordpacks/`

Requirements:

- Preserve non-empty words/phrases and comments where practical.
- Include non-English packs such as German, Dutch, Czech, Persian/Farsi, Harry Potter, adult/custom packs, and English variants.
- UI remains English only; wordpack names can be filenames or simple labels derived from filenames.
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
- Normalize/cache images server-side if implemented; generated cache files live outside source assets or in a documented cache directory.
- Expose only safe image ids to clients, not arbitrary filesystem paths.
