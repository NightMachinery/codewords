export type ThemeId = 'dark' | 'light' | 'matrix';

export interface ThemeDescriptor {
  id: ThemeId;
  label: string;
  mode: 'dark' | 'light';
}

export const THEMES: ThemeDescriptor[] = [
  { id: 'dark', label: 'Dark', mode: 'dark' },
  { id: 'light', label: 'Light', mode: 'light' },
  { id: 'matrix', label: 'Matrix', mode: 'dark' },
];

export const themeIds: ThemeId[] = THEMES.map((theme) => theme.id);
export const darkModeThemes: ThemeDescriptor[] = THEMES.filter((theme) => theme.mode === 'dark');
export const lightModeThemes: ThemeDescriptor[] = THEMES.filter((theme) => theme.mode === 'light');

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
