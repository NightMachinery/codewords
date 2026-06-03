export type ThemeId = 'dark' | 'light' | 'matrix' | 'solarized-light' | 'solarized-dark' | 'spiderman';

export interface ThemeDescriptor {
  id: ThemeId;
  label: string;
  mode: 'dark' | 'light';
}

export const THEMES: ThemeDescriptor[] = [
  { id: 'dark', label: 'Dark', mode: 'dark' },
  { id: 'light', label: 'Light', mode: 'light' },
  { id: 'matrix', label: 'Matrix', mode: 'dark' },
  { id: 'solarized-dark', label: 'Solarized Dark', mode: 'dark' },
  { id: 'solarized-light', label: 'Solarized Light', mode: 'light' },
  { id: 'spiderman', label: 'Spider-Man', mode: 'dark' },
];

export const themeIds: ThemeId[] = THEMES.map((theme) => theme.id);
export const darkModeThemes: ThemeDescriptor[] = THEMES.filter((theme) => theme.mode === 'dark');
export const lightModeThemes: ThemeDescriptor[] = THEMES.filter((theme) => theme.mode === 'light');

/** A linear-RGB triple (each component 0..1) for the aurora WebGL shader. */
export type Rgb = readonly [number, number, number];
export type AuroraShaderVariant = 'aurora' | 'clean-fire' | 'campfire';

/**
 * Per-theme palette for the landing-hero aurora shader. `sky*` are the vertical
 * background gradient stops; `ribbon*` are the three animated aurora/fire colors;
 * `shader` chooses which procedural background renderer the theme uses.
 */
export interface AuroraPalette {
  shader: AuroraShaderVariant;
  skyTop: Rgb;
  skyMid: Rgb;
  skyLow: Rgb;
  ribbonA: Rgb;
  ribbonB: Rgb;
  ribbonC: Rgb;
}

export const auroraPalettes: Record<ThemeId, AuroraPalette> = {
  // Dark: the original deep-navy sky with emerald/cyan/violet ribbons.
  dark: {
    shader: 'aurora',
    skyTop: [0.004, 0.01, 0.026],
    skyMid: [0.01, 0.022, 0.052],
    skyLow: [0.015, 0.03, 0.06],
    ribbonA: [0.22, 0.92, 0.62],
    ribbonB: [0.2, 0.7, 0.96],
    ribbonC: [0.48, 0.3, 0.82],
  },
  // Light: a pale warm sky with literal flame colors for the light-theme shader path.
  light: {
    shader: 'clean-fire',
    skyTop: [0.99, 0.94, 0.84],
    skyMid: [0.98, 0.86, 0.68],
    skyLow: [0.93, 0.72, 0.46],
    ribbonA: [1.0, 0.82, 0.22],
    ribbonB: [1.0, 0.42, 0.08],
    ribbonC: [0.9, 0.14, 0.18],
  },
  // Matrix: near-black green sky with phosphor-green ribbons.
  matrix: {
    shader: 'aurora',
    skyTop: [0.004, 0.03, 0.012],
    skyMid: [0.01, 0.05, 0.022],
    skyLow: [0.012, 0.07, 0.03],
    ribbonA: [0.18, 0.98, 0.42],
    ribbonB: [0.3, 0.85, 0.3],
    ribbonC: [0.1, 0.7, 0.32],
  },
  // Solarized Dark: base03 teal-grey sky with solarized blue/cyan/violet ribbons.
  'solarized-dark': {
    shader: 'aurora',
    skyTop: [0.0, 0.12, 0.15],
    skyMid: [0.0, 0.17, 0.21],
    skyLow: [0.03, 0.21, 0.26],
    ribbonA: [0.15, 0.55, 0.82],
    ribbonB: [0.16, 0.63, 0.6],
    ribbonC: [0.42, 0.44, 0.77],
  },
  // Solarized Light: cream sky with Solarized-compatible amber/orange fire accents.
  'solarized-light': {
    shader: 'clean-fire',
    skyTop: [0.99, 0.96, 0.89],
    skyMid: [0.96, 0.88, 0.7],
    skyLow: [0.9, 0.76, 0.48],
    ribbonA: [0.98, 0.72, 0.18],
    ribbonB: [0.9, 0.38, 0.06],
    ribbonC: [0.78, 0.12, 0.22],
  },
  // Spider-Man: dark navy sky with red/blue/white hero-color ribbons.
  spiderman: {
    shader: 'aurora',
    skyTop: [0.02, 0.02, 0.08],
    skyMid: [0.04, 0.04, 0.13],
    skyLow: [0.07, 0.06, 0.16],
    ribbonA: [0.86, 0.13, 0.16],
    ribbonB: [0.16, 0.32, 0.85],
    ribbonC: [0.95, 0.95, 0.98],
  },
};

export function auroraPaletteFor(id: ThemeId): AuroraPalette {
  return auroraPalettes[id] ?? auroraPalettes.dark;
}

/**
 * Background for the end-game memory-capture image. `gradient` is the CSS background
 * painted on the capture element; `solid` is the flat matte color handed to
 * html-to-image (used behind any transparency). Card colors stay team-semantic
 * (see cardCaptureColors in memoryCapture.ts) and are intentionally not themed.
 */
export interface CaptureBackground {
  solid: string;
  gradient: string;
}

export const captureBackgrounds: Record<ThemeId, CaptureBackground> = {
  dark: {
    solid: '#07111f',
    gradient:
      'radial-gradient(circle at 15% 8%, rgba(16,185,129,0.24), transparent 32%), radial-gradient(circle at 82% 16%, rgba(59,130,246,0.2), transparent 34%), linear-gradient(135deg, #07111f, #0f172a 52%, #07101a)',
  },
  light: {
    solid: '#eef2f8',
    gradient:
      'radial-gradient(circle at 15% 8%, rgba(16,185,129,0.16), transparent 32%), radial-gradient(circle at 82% 16%, rgba(59,130,246,0.14), transparent 34%), linear-gradient(135deg, #f3f6fb, #e7edf6 52%, #eef2f8)',
  },
  matrix: {
    solid: '#04140b',
    gradient:
      'radial-gradient(circle at 15% 8%, rgba(34,200,90,0.28), transparent 32%), radial-gradient(circle at 82% 16%, rgba(34,160,70,0.2), transparent 34%), linear-gradient(135deg, #04140b, #07210f 52%, #03100a)',
  },
  'solarized-dark': {
    solid: '#04222b',
    gradient:
      'radial-gradient(circle at 15% 8%, rgba(38,139,210,0.24), transparent 32%), radial-gradient(circle at 82% 16%, rgba(42,161,152,0.2), transparent 34%), linear-gradient(135deg, #04222b, #073642 52%, #032029)',
  },
  'solarized-light': {
    solid: '#fdf6e3',
    gradient:
      'radial-gradient(circle at 15% 8%, rgba(133,153,0,0.16), transparent 32%), radial-gradient(circle at 82% 16%, rgba(38,139,210,0.14), transparent 34%), linear-gradient(135deg, #fdf6e3, #eee8d5 52%, #fdf6e3)',
  },
  spiderman: {
    solid: '#0a0a18',
    gradient:
      'radial-gradient(circle at 15% 8%, rgba(220,38,38,0.28), transparent 32%), radial-gradient(circle at 82% 16%, rgba(37,99,235,0.24), transparent 34%), linear-gradient(135deg, #0a0a18, #12122a 52%, #08081a)',
  },
};

export function captureBackgroundFor(id: ThemeId): CaptureBackground {
  return captureBackgrounds[id] ?? captureBackgrounds.dark;
}

export function themeMode(id: ThemeId): 'dark' | 'light' {
  return THEMES.find((theme) => theme.id === id)?.mode ?? 'dark';
}

export function isThemeId(value: unknown): value is ThemeId {
  return typeof value === 'string' && themeIds.includes(value as ThemeId);
}

/**
 * `auto` follows the OS `prefers-color-scheme`, picking `darkTheme` or `lightTheme`.
 * When `auto` is off, `manual` is used directly.
 */
export interface ThemePreferences {
  auto: boolean;
  manual: ThemeId;
  darkTheme: ThemeId;
  lightTheme: ThemeId;
}

export const themePreferencesStorageKey = 'codewords.themePreferences';

export const defaultThemePreferences: ThemePreferences = {
  auto: false,
  manual: 'dark',
  darkTheme: 'dark',
  lightTheme: 'light',
};

export interface ThemeStorage {
  getItem(key: string): string | null;
  setItem(key: string, value: string): void;
}

export function readThemePreferences(storage: Pick<ThemeStorage, 'getItem'>): ThemePreferences {
  const raw = storage.getItem(themePreferencesStorageKey);
  if (!raw) return { ...defaultThemePreferences };
  try {
    const parsed = JSON.parse(raw) as Partial<ThemePreferences>;
    return {
      auto: typeof parsed.auto === 'boolean' ? parsed.auto : defaultThemePreferences.auto,
      manual: isThemeId(parsed.manual) ? parsed.manual : defaultThemePreferences.manual,
      darkTheme: isThemeId(parsed.darkTheme) ? parsed.darkTheme : defaultThemePreferences.darkTheme,
      lightTheme: isThemeId(parsed.lightTheme) ? parsed.lightTheme : defaultThemePreferences.lightTheme,
    };
  } catch {
    return { ...defaultThemePreferences };
  }
}

export function writeThemePreferences(storage: Pick<ThemeStorage, 'setItem'>, preferences: ThemePreferences): void {
  storage.setItem(themePreferencesStorageKey, JSON.stringify(preferences));
}

/** Resolve the theme that should be shown, given the saved preferences and the OS dark-mode state. */
export function resolveTheme(preferences: ThemePreferences, prefersDark: boolean): ThemeId {
  if (!preferences.auto) return preferences.manual;
  return prefersDark ? preferences.darkTheme : preferences.lightTheme;
}

/** Apply a theme by setting `data-theme` on the document root. No-op outside the browser. */
export function applyTheme(id: ThemeId): void {
  if (typeof document === 'undefined') return;
  document.documentElement.setAttribute('data-theme', id);
}

/** True when the OS currently prefers a dark color scheme. Safe outside the browser. */
export function prefersDarkScheme(): boolean {
  if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') return true;
  return window.matchMedia('(prefers-color-scheme: dark)').matches;
}

/**
 * Subscribe to OS dark-mode changes. Calls `onChange(prefersDark)` whenever the system
 * preference flips. Returns an unsubscribe function. No-op outside the browser.
 */
export function watchColorScheme(onChange: (prefersDark: boolean) => void): () => void {
  if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') return () => {};
  const media = window.matchMedia('(prefers-color-scheme: dark)');
  const listener = (event: MediaQueryListEvent) => onChange(event.matches);
  media.addEventListener('change', listener);
  return () => media.removeEventListener('change', listener);
}
