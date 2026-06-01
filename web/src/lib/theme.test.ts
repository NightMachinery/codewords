import { describe, expect, it } from 'vitest';

import {
  defaultThemePreferences,
  readThemePreferences,
  resolveTheme,
  themePreferencesStorageKey,
  THEMES,
  writeThemePreferences,
  type ThemePreferences,
  type ThemeStorage,
} from './theme';

class MemoryStorage implements ThemeStorage {
  private values = new Map<string, string>();

  getItem(key: string): string | null {
    return this.values.get(key) ?? null;
  }

  setItem(key: string, value: string): void {
    this.values.set(key, value);
  }
}

describe('theme preferences', () => {
  it('returns defaults when nothing is stored', () => {
    const storage = new MemoryStorage();
    expect(readThemePreferences(storage)).toEqual(defaultThemePreferences);
  });

  it('round-trips written preferences', () => {
    const storage = new MemoryStorage();
    const prefs: ThemePreferences = { auto: true, manual: 'matrix', darkTheme: 'matrix', lightTheme: 'light' };
    writeThemePreferences(storage, prefs);
    expect(readThemePreferences(storage)).toEqual(prefs);
  });

  it('accepts the Solarized and Spider-Man theme ids', () => {
    const storage = new MemoryStorage();
    const prefs: ThemePreferences = { auto: true, manual: 'spiderman', darkTheme: 'solarized-dark', lightTheme: 'solarized-light' };
    writeThemePreferences(storage, prefs);
    expect(readThemePreferences(storage)).toEqual(prefs);
  });

  it('falls back to defaults for invalid theme ids', () => {
    const storage = new MemoryStorage();
    storage.setItem(
      themePreferencesStorageKey,
      JSON.stringify({ auto: 'yes', manual: 'neon', darkTheme: 'dark', lightTheme: 42 }),
    );
    expect(readThemePreferences(storage)).toEqual({
      auto: defaultThemePreferences.auto,
      manual: defaultThemePreferences.manual,
      darkTheme: 'dark',
      lightTheme: defaultThemePreferences.lightTheme,
    });
  });

  it('falls back to defaults for corrupt JSON', () => {
    const storage = new MemoryStorage();
    storage.setItem(themePreferencesStorageKey, '{not json');
    expect(readThemePreferences(storage)).toEqual(defaultThemePreferences);
  });
});

describe('resolveTheme', () => {
  it('uses the manual theme when auto is off, ignoring the OS preference', () => {
    const prefs: ThemePreferences = { auto: false, manual: 'matrix', darkTheme: 'dark', lightTheme: 'light' };
    expect(resolveTheme(prefs, true)).toBe('matrix');
    expect(resolveTheme(prefs, false)).toBe('matrix');
  });

  it('picks the dark theme when auto is on and the OS prefers dark', () => {
    const prefs: ThemePreferences = { auto: true, manual: 'light', darkTheme: 'matrix', lightTheme: 'light' };
    expect(resolveTheme(prefs, true)).toBe('matrix');
  });

  it('picks the light theme when auto is on and the OS prefers light', () => {
    const prefs: ThemePreferences = { auto: true, manual: 'matrix', darkTheme: 'matrix', lightTheme: 'light' };
    expect(resolveTheme(prefs, false)).toBe('light');
  });

  it('resolves to the chosen Solarized themes per OS preference', () => {
    const prefs: ThemePreferences = { auto: true, manual: 'dark', darkTheme: 'solarized-dark', lightTheme: 'solarized-light' };
    expect(resolveTheme(prefs, true)).toBe('solarized-dark');
    expect(resolveTheme(prefs, false)).toBe('solarized-light');
  });
});

describe('THEMES catalog', () => {
  it('exposes the six themes with valid modes', () => {
    expect(THEMES.map((t) => t.id)).toEqual([
      'dark',
      'light',
      'matrix',
      'solarized-dark',
      'solarized-light',
      'spiderman',
    ]);
    for (const theme of THEMES) {
      expect(['dark', 'light']).toContain(theme.mode);
    }
  });
});
