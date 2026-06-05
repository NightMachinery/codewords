# Frontend Theming

Codewords ships twelve themes — **Dark** (the original look), **Light**, **Matrix**, **Solarized
Dark**, **Solarized Light**, **Spider-Man**, **Dracula**, **Glitch**, **Christmas Cozy**,
**Christmas Snow**, **Christmas Candy**, and **Blood** — with optional automatic switching that
follows the operating system's dark-mode preference.

## How themes are applied

Styling is Tailwind v4. Every color utility (e.g. `bg-slate-950`, `text-emerald-300`) compiles to
a CSS variable reference such as `var(--color-slate-950)`. Re-skinning the whole app is therefore
just a matter of overriding those variables — no component edits are required.

`web/src/style.css` defines `[data-theme="…"]` blocks that override the **neutral chrome**
(`slate`, `zinc`) and the **emerald/teal/cyan accent** tokens per theme:

- **Dark** — the Tailwind defaults, so it needs no overrides (it is the baseline).
- **Light** — inverts the slate/zinc lightness ramp (dark text on light surfaces) and darkens the
  emerald/cyan/teal accents for contrast; sets `color-scheme: light`. (Cyan/teal must be overridden
  too, otherwise accent text like the Join button stays near-white and unreadable on light.)
- **Matrix** — recolors the slate ramp to near-black greens with a bright phosphor-green accent;
  sets `color-scheme: dark`.
- **Solarized Dark / Light** — Ethan Schoonover's base03↔base3 ramp with a Solarized-green accent;
  the light variant darkens cyan/teal for contrast like Light does.
- **Spider-Man** — deep navy-blue surfaces with a bright Spidey-red accent and web-blue secondary
  accents; `color-scheme: dark`.
- **Dracula** — editor-purple surfaces based on the Dracula palette, with green controls and
  purple, pink, and cyan shader accents; `color-scheme: dark`.
- **Glitch** — near-black CRT/digital-corruption surfaces with electric cyan, toxic green,
  magenta, and violet signal accents; `color-scheme: dark`.
- **Christmas Cozy** — evergreen night surfaces with warm gold controls, holly red, and mint snow
  accents; `color-scheme: dark`.
- **Christmas Snow** — frosted blue-white surfaces with holly green controls and restrained red
  accents; `color-scheme: light`.
- **Christmas Candy** — peppermint cream and mint surfaces with candy-cane red controls and green
  secondary accents; `color-scheme: light`.
- **Blood** — elegant noir burgundy surfaces with polished crimson controls, oxblood highlights,
  and custom dripping blood shaders; `color-scheme: dark`.

Team colors (`blue` = Libertarians, `red` = Monarchists, plus Unity/Monality custom accents) and warning `amber` are intentionally left semantically stable so players can recognize card allegiance in every theme. Card, counter, and clue-badge foregrounds are contrast-aware rather than fixed near-white: helpers in `gameplay.ts` blend the team tint over the active board surface and choose a dark or light OKLCH foreground. This keeps low-alpha purple Monality surfaces readable on light themes while preserving light text on dark saturated cards. The light themes still darken the low `amber` text shades so warm labels stay readable.

The active theme is selected by setting `data-theme` on `<html>` via `applyTheme()` in
`web/src/lib/theme.ts`. It is applied in `web/src/main.ts` before the app mounts so the correct
theme paints on the first frame (no flash). The full theme list lives in `THEMES` in `theme.ts`;
adding a theme is `THEMES` + an `auroraPalettes` entry + a `[data-theme]` block in `style.css`.

## Preferences and persistence

`web/src/lib/theme.ts` owns the theme model, kept separate from gameplay preferences so that a
moderator-pushed theme (see below) can never overwrite a saved preference:

```ts
interface ThemePreferences {
  auto: boolean;       // follow the OS prefers-color-scheme
  manual: ThemeId;     // used when auto is off
  darkTheme: ThemeId;  // used when auto is on and the OS prefers dark   (default 'dark')
  lightTheme: ThemeId; // used when auto is on and the OS prefers light  (default 'light')
}
```

Preferences are stored in `localStorage` under `codewords.themePreferences` via
`readThemePreferences` / `writeThemePreferences` (validated with fallbacks, matching the
`gameplay.ts` pattern). `resolveTheme(prefs, prefersDark)` returns the theme to show, and
`watchColorScheme` re-resolves when the OS preference flips while `auto` is on.

## UI

The theme picker is a single shared component, `web/src/lib/ThemeMenu.svelte` — an icon-only button
(the icon reflects the active theme, or `SunMoon` when auto is on) that opens a popover with the
"auto" checkbox plus either per-mode dark/light selects (auto on) or a single-click theme list (auto
off). It owns the theme state, persistence, OS-preference watching, and `applyTheme`, and exposes
bindable `effectiveThemeId` / `sessionOverride` / `themePreferences` props.

- **Room** (`web/src/pages/RoomPage.svelte`): `<ThemeMenu>` sits in the top status row and renders
  its popover as a fixed, frontmost overlay so the nav row cannot clip it. The Local Settings panel
  mirrors the same theme preferences as dropdown controls; changing either surface persists the
  local preference and clears any session-only moderator override. Moderators also see the "push
  theme to everyone" button there. Destructive icon-only controls such as the active-match restart
  button use local theme-aware styling for their red foregrounds, because global red token remapping
  would interfere with team/error semantics.
- **Home page** (`web/src/pages/HomePage.svelte`): `<ThemeMenu>` sits top-right in the nav. The
  landing hero is theme-aware — its room-entry card and controls use theme tokens, and the animated
  background changes by theme-selected shader variant. Dark themes render cool aurora curtains.
  Light uses the detailed `campfire` variant with uneven lower flame mass, torn rising licks,
  sparse sparks, and faint smoke/haze. Solarized Light uses the cleaner existing fire variant with
  smoother flame tongues and ember wisps. Dracula uses separate `dracula-home`, `dracula-board`,
  and `dracula-card` shader variants, selected through the `surface` prop. Glitch likewise uses
  `glitch-home`, `glitch-board`, and `glitch-card` for CRT scanlines, RGB tearing, broken slices,
  and intermittent block noise. Blood uses `blood-home`, `blood-board`, and `blood-card` for slow
  crimson curtains, pooled board motion, and glossy card rivulets/droplets. The Christmas themes
  each use their own `home`, `board`, and `card` shader variants for cozy string lights and snow,
  snowy frost veils, or peppermint stripe motion.
  Christmas Snow keeps its board/card shader readable but visibly festive: icy-blue drift bands, sparse moving flakes/crystals, and light holly accents avoid the old gray wash. Christmas Candy uses stronger peppermint bands and mint/pink sugar highlights, with light-theme blend modes chosen to keep those colors bright instead of multiplying into gray.
  The shader's sky/ribbon colors and variant are fed from a
  per-theme `AuroraPalette` in `theme.ts` (`AuroraBackground` takes `theme` and `surface` props,
  updates uniforms reactively, and swaps complete fragment shader sources when the selected
  variant changes); the CSS fallback and base color mirror those palettes via `[data-theme]`
  selectors.
- **Room board** (`web/src/lib/BoardGrid.svelte`): themes that define `surfaceShaders.board` and
  `surfaceShaders.card` render one shared board shader behind the grid and one distinct card-sheen
  shader overlay above the grid, both decorative and pointer-events-free. Dracula, Glitch, Blood, and the
  Christmas themes use this path. Visible civilian and assassin cards use explicit semantic inline chrome above the generic shader card backing, so neutral and assassin cards remain distinct in shader themes; unrevealed cards keep the shader-themed backing. Dracula still gives revealed civilian cards an extra stronger amber surface after its generic card backing. Christmas Snow uses a lower-opacity board shader, nearly transparent card sheen, and stronger frosted card backings so pale surfaces do not wash out text. Capture mode omits live shader layers so memory exports stay stable.

## Moderator push (session only)

Moderators see a **"Push current theme to everyone"** button in the room's Local Settings. It sends
a `forceTheme` WebSocket command; the server gates it to moderators and broadcasts a `themeForced`
message (see `http-and-realtime.md`). Receivers apply the pushed theme **for the current session
only** — it is never written to `localStorage`, so reloading restores each player's own saved theme.
Changing the local theme settings clears the moderator override and returns control to the user.

This deliberately mirrors the older `forceBoardLayout` / `boardLayoutForced` flow, but is
session-only rather than persisted.

## Memory capture

The end-game shareable PNG follows the active theme. `html-to-image` inlines computed styles, so the
themed `var(--color-*)` utilities inside the capture (panels, text) are captured correctly; only the
backdrop needed wiring. `captureBackgrounds` in `theme.ts` provides a per-theme `{ solid, gradient }`
— the gradient paints the capture element and the solid is the matte handed to `html-to-image`'s
`backgroundColor` (threaded through `downloadMemoryCapture`). Card colors stay team-semantic
(`cardCaptureColors` in `memoryCapture.ts`) and are deliberately not themed.
