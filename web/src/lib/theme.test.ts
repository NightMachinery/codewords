import { describe, expect, it } from 'vitest';

import {
  auroraPalettes,
  auroraPaletteFor,
  defaultThemePreferences,
  darkModeThemes,
  lightModeThemes,
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

  it('accepts the Solarized, Spider-Man, Dracula, Glitch, and Christmas theme ids', () => {
    const storage = new MemoryStorage();
    const prefs: ThemePreferences = { auto: true, manual: 'christmas-candy', darkTheme: 'christmas-cozy', lightTheme: 'christmas-snow' };
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
  it('exposes the eleven themes with valid modes', () => {
    expect(THEMES.map((t) => t.id)).toEqual([
      'dark',
      'light',
      'matrix',
      'solarized-dark',
      'solarized-light',
      'spiderman',
      'dracula',
      'glitch',
      'christmas-cozy',
      'christmas-snow',
      'christmas-candy',
    ]);
    for (const theme of THEMES) {
      expect(['dark', 'light']).toContain(theme.mode);
    }
    expect(darkModeThemes.map((theme) => theme.id)).toContain('dracula');
    expect(darkModeThemes.map((theme) => theme.id)).toContain('glitch');
    expect(darkModeThemes.map((theme) => theme.id)).toContain('christmas-cozy');
    expect(lightModeThemes.map((theme) => theme.id)).toEqual(expect.arrayContaining(['christmas-snow', 'christmas-candy']));
  });
});

describe('aurora shader variants', () => {
  it('assigns a valid shader variant to every theme palette', () => {
    const validVariants = [
      'aurora',
      'clean-fire',
      'campfire',
      'dracula-home',
      'dracula-board',
      'dracula-card',
      'glitch-home',
      'glitch-board',
      'glitch-card',
      'christmas-cozy-home',
      'christmas-cozy-board',
      'christmas-cozy-card',
      'christmas-snow-home',
      'christmas-snow-board',
      'christmas-snow-card',
      'christmas-candy-home',
      'christmas-candy-board',
      'christmas-candy-card',
    ];
    for (const theme of THEMES) {
      expect(validVariants).toContain(auroraPalettes[theme.id].shader);
    }
  });

  it('uses surface-specific Dracula shader variants', () => {
    expect(auroraPaletteFor('dracula', 'home').shader).toBe('dracula-home');
    expect(auroraPaletteFor('dracula', 'board').shader).toBe('dracula-board');
    expect(auroraPaletteFor('dracula', 'card').shader).toBe('dracula-card');
  });

  it('uses surface-specific Glitch shader variants', () => {
    expect(auroraPaletteFor('glitch', 'home').shader).toBe('glitch-home');
    expect(auroraPaletteFor('glitch', 'board').shader).toBe('glitch-board');
    expect(auroraPaletteFor('glitch', 'card').shader).toBe('glitch-card');
  });

  it('uses surface-specific Christmas shader variants', () => {
    for (const id of ['christmas-cozy', 'christmas-snow', 'christmas-candy'] as const) {
      expect(auroraPaletteFor(id, 'home').shader).toBe(`${id}-home`);
      expect(auroraPaletteFor(id, 'board').shader).toBe(`${id}-board`);
      expect(auroraPaletteFor(id, 'card').shader).toBe(`${id}-card`);
    }
  });

  it('uses campfire for Light, clean fire for Solarized Light, and aurora for base dark themes', () => {
    expect(auroraPalettes.light.shader).toBe('campfire');
    expect(auroraPalettes['solarized-light'].shader).toBe('clean-fire');
    expect(auroraPalettes.dark.shader).toBe('aurora');
    expect(auroraPalettes.matrix.shader).toBe('aurora');
    expect(auroraPalettes['solarized-dark'].shader).toBe('aurora');
    expect(auroraPalettes.spiderman.shader).toBe('aurora');
  });
});
