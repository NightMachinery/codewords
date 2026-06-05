export type ThemeId =
  | 'dark'
  | 'light'
  | 'matrix'
  | 'solarized-light'
  | 'solarized-dark'
  | 'spiderman'
  | 'dracula'
  | 'glitch'
  | 'christmas-cozy'
  | 'christmas-snow'
  | 'christmas-candy'
  | 'blood';

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
  { id: 'dracula', label: 'Dracula', mode: 'dark' },
  { id: 'glitch', label: 'Glitch', mode: 'dark' },
  { id: 'christmas-cozy', label: 'Christmas Cozy', mode: 'dark' },
  { id: 'christmas-snow', label: 'Christmas Snow', mode: 'light' },
  { id: 'christmas-candy', label: 'Christmas Candy', mode: 'light' },
  { id: 'blood', label: 'Blood', mode: 'dark' },
];

export const themeIds: ThemeId[] = THEMES.map((theme) => theme.id);
export const darkModeThemes: ThemeDescriptor[] = THEMES.filter((theme) => theme.mode === 'dark');
export const lightModeThemes: ThemeDescriptor[] = THEMES.filter((theme) => theme.mode === 'light');

/** A linear-RGB triple (each component 0..1) for the aurora WebGL shader. */
export type Rgb = readonly [number, number, number];
export type ThemeShaderSurface = 'home' | 'board' | 'card';
export type AuroraShaderVariant =
  | 'aurora'
  | 'clean-fire'
  | 'campfire'
  | 'dracula-home'
  | 'dracula-board'
  | 'dracula-card'
  | 'glitch-home'
  | 'glitch-board'
  | 'glitch-card'
  | 'christmas-cozy-home'
  | 'christmas-cozy-board'
  | 'christmas-cozy-card'
  | 'christmas-snow-home'
  | 'christmas-snow-board'
  | 'christmas-snow-card'
  | 'christmas-candy-home'
  | 'christmas-candy-board'
  | 'christmas-candy-card'
  | 'blood-home'
  | 'blood-board'
  | 'blood-card';

/**
 * Per-theme palette for the landing-hero aurora shader. `sky*` are the vertical
 * background gradient stops; `ribbon*` are the three animated aurora/fire colors;
 * `shader` chooses which procedural background renderer the theme uses.
 */
export interface AuroraPalette {
  shader: AuroraShaderVariant;
  surfaceShaders?: Partial<Record<ThemeShaderSurface, AuroraShaderVariant>>;
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
    shader: 'campfire',
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
  // Dracula: deep editor purple with cyan/pink/purple shader surfaces.
  dracula: {
    shader: 'dracula-home',
    surfaceShaders: {
      home: 'dracula-home',
      board: 'dracula-board',
      card: 'dracula-card',
    },
    skyTop: [0.08, 0.085, 0.13],
    skyMid: [0.12, 0.12, 0.19],
    skyLow: [0.17, 0.18, 0.25],
    ribbonA: [0.74, 0.58, 0.98],
    ribbonB: [1.0, 0.47, 0.78],
    ribbonC: [0.55, 0.91, 0.99],
  },
  // Glitch: near-black CRT corruption with cyan/green/magenta signal channels.
  glitch: {
    shader: 'glitch-home',
    surfaceShaders: {
      home: 'glitch-home',
      board: 'glitch-board',
      card: 'glitch-card',
    },
    skyTop: [0.002, 0.004, 0.012],
    skyMid: [0.008, 0.014, 0.03],
    skyLow: [0.012, 0.018, 0.026],
    ribbonA: [0.0, 0.94, 1.0],
    ribbonB: [0.48, 1.0, 0.08],
    ribbonC: [1.0, 0.08, 0.72],
  },
  // Christmas Cozy: evergreen night with warm gold lights, holly red, and soft snow.
  'christmas-cozy': {
    shader: 'christmas-cozy-home',
    surfaceShaders: {
      home: 'christmas-cozy-home',
      board: 'christmas-cozy-board',
      card: 'christmas-cozy-card',
    },
    skyTop: [0.006, 0.024, 0.02],
    skyMid: [0.02, 0.08, 0.055],
    skyLow: [0.05, 0.12, 0.07],
    ribbonA: [1.0, 0.78, 0.28],
    ribbonB: [0.9, 0.08, 0.1],
    ribbonC: [0.36, 0.9, 0.56],
  },
  // Christmas Snow: bright winter sky with frosted blue-white surfaces and holly accents.
  'christmas-snow': {
    shader: 'christmas-snow-home',
    surfaceShaders: {
      home: 'christmas-snow-home',
      board: 'christmas-snow-board',
      card: 'christmas-snow-card',
    },
    skyTop: [0.86, 0.94, 1.0],
    skyMid: [0.94, 0.97, 1.0],
    skyLow: [0.98, 0.99, 1.0],
    ribbonA: [0.12, 0.45, 0.24],
    ribbonB: [0.86, 0.08, 0.12],
    ribbonC: [0.38, 0.66, 0.95],
  },
  // Christmas Candy: mint and cream surfaces with peppermint red/green holiday motion.
  'christmas-candy': {
    shader: 'christmas-candy-home',
    surfaceShaders: {
      home: 'christmas-candy-home',
      board: 'christmas-candy-board',
      card: 'christmas-candy-card',
    },
    skyTop: [0.98, 0.99, 0.94],
    skyMid: [0.95, 1.0, 0.96],
    skyLow: [1.0, 0.94, 0.9],
    ribbonA: [0.92, 0.03, 0.08],
    ribbonB: [0.0, 0.58, 0.25],
    ribbonC: [1.0, 0.78, 0.82],
  },

  // Blood: elegant noir burgundy with liquid crimson shader surfaces.
  blood: {
    shader: 'blood-home',
    surfaceShaders: {
      home: 'blood-home',
      board: 'blood-board',
      card: 'blood-card',
    },
    skyTop: [0.025, 0.006, 0.01],
    skyMid: [0.075, 0.012, 0.02],
    skyLow: [0.13, 0.018, 0.026],
    ribbonA: [0.86, 0.02, 0.06],
    ribbonB: [0.42, 0.0, 0.02],
    ribbonC: [1.0, 0.22, 0.18],
  },
};

export function auroraPaletteFor(id: ThemeId, surface: ThemeShaderSurface = 'home'): AuroraPalette {
  const palette = auroraPalettes[id] ?? auroraPalettes.dark;
  return { ...palette, shader: palette.surfaceShaders?.[surface] ?? palette.shader };
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
  dracula: {
    solid: '#282a36',
    gradient:
      'radial-gradient(circle at 15% 8%, rgba(189,147,249,0.28), transparent 32%), radial-gradient(circle at 82% 16%, rgba(255,121,198,0.22), transparent 34%), radial-gradient(circle at 48% 88%, rgba(139,233,253,0.18), transparent 38%), linear-gradient(135deg, #282a36, #1f2130 52%, #171923)',
  },
  glitch: {
    solid: '#03040a',
    gradient:
      'radial-gradient(circle at 12% 10%, rgba(0,240,255,0.24), transparent 30%), radial-gradient(circle at 84% 14%, rgba(255,20,170,0.18), transparent 32%), radial-gradient(circle at 52% 86%, rgba(115,255,24,0.14), transparent 36%), linear-gradient(135deg, #03040a, #07101b 52%, #020308)',
  },
  'christmas-cozy': {
    solid: '#06140d',
    gradient:
      'radial-gradient(circle at 14% 10%, rgba(250,204,21,0.22), transparent 30%), radial-gradient(circle at 84% 15%, rgba(220,38,38,0.18), transparent 32%), radial-gradient(circle at 48% 88%, rgba(74,222,128,0.14), transparent 38%), linear-gradient(135deg, #06140d, #0b2417 52%, #08110c)',
  },
  'christmas-snow': {
    solid: '#edf6fb',
    gradient:
      'radial-gradient(circle at 12% 10%, rgba(34,197,94,0.16), transparent 30%), radial-gradient(circle at 84% 15%, rgba(220,38,38,0.12), transparent 32%), radial-gradient(circle at 48% 88%, rgba(125,211,252,0.22), transparent 38%), linear-gradient(135deg, #f8fbff, #e9f4fb 52%, #dfeaf5)',
  },
  'christmas-candy': {
    solid: '#fff7f3',
    gradient:
      'radial-gradient(circle at 12% 10%, rgba(239,68,68,0.16), transparent 30%), radial-gradient(circle at 84% 15%, rgba(34,197,94,0.16), transparent 32%), radial-gradient(circle at 48% 88%, rgba(251,113,133,0.14), transparent 38%), linear-gradient(135deg, #fffafa, #effdf5 52%, #fff1f2)',
  },

  blood: {
    solid: '#120306',
    gradient:
      'radial-gradient(circle at 14% 10%, rgba(185,28,28,0.26), transparent 30%), radial-gradient(circle at 82% 18%, rgba(127,29,29,0.2), transparent 34%), radial-gradient(circle at 48% 88%, rgba(239,68,68,0.14), transparent 38%), linear-gradient(135deg, #120306, #2a060b 52%, #080103)',
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
