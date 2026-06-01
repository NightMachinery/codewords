# Frontend Theming

Codewords ships six themes — **Dark** (the original look), **Light**, **Matrix**, **Solarized
Dark**, **Solarized Light**, and **Spider-Man** — with optional automatic switching that follows
the operating system's dark-mode preference.

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

Team colors (`blue` = Libertarians, `red` = Monarchists) and warning `amber` are intentionally left
at their Tailwind defaults so they stay recognizable in every theme.

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

- **Room** (`web/src/pages/RoomPage.svelte`): `<ThemeMenu>` sits in the top status row. The Local
  Settings panel keeps only the moderator "push theme to everyone" button.
- **Home page** (`web/src/pages/HomePage.svelte`): `<ThemeMenu>` sits top-right in the nav. The
  landing hero is theme-aware — its room-entry card and controls use theme tokens, and the animated
  aurora background is recolored per theme. The aurora shader's sky/ribbon colors are uniforms fed
  from a per-theme `AuroraPalette` in `theme.ts` (`AuroraBackground` takes a `theme` prop and updates
  the uniforms reactively); the CSS fallback and base color mirror those palettes via `[data-theme]`
  selectors.

## Moderator push (session only)

Moderators see a **"Push current theme to everyone"** button in the room's Local Settings. It sends
a `forceTheme` WebSocket command; the server gates it to moderators and broadcasts a `themeForced`
message (see `http-and-realtime.md`). Receivers apply the pushed theme **for the current session
only** — it is never written to `localStorage`, so reloading restores each player's own saved theme.
Changing the local theme settings clears the moderator override and returns control to the user.

This deliberately mirrors the older `forceBoardLayout` / `boardLayoutForced` flow, but is
session-only rather than persisted.
