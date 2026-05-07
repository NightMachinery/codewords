import { describe, expect, it } from 'vitest';

import { defaultSettings, type Settings } from './api';
import {
  applySettingsProfile,
  exportSettingsProfileJson5,
  parseSettingsProfileJson5,
  readSavedProfiles,
  writeSavedProfiles,
  settingsProfilesStorageKey,
  type SettingsProfile,
} from './settingsProfiles';

class MemoryStorage {
  values = new Map<string, string>();
  getItem(key: string): string | null {
    return this.values.get(key) ?? null;
  }
  setItem(key: string, value: string): void {
    this.values.set(key, value);
  }
}

const current: Settings = { ...defaultSettings, wordpackId: 'english', imageCardCount: 0, blackCards: 1, totalCards: 25 };

describe('settings profiles', () => {
  it('applies only known partial settings and ignores extra fields', () => {
    const next = applySettingsProfile(current, {
      settings: {
        imageCardCount: 2,
        mixedImageOrderFirst: true,
        blackCards: 2,
        totalCards: 26,
        unknownField: 'ignored',
      } as Partial<Settings> & Record<string, unknown>,
    });

    expect(next).toMatchObject({ imageCardCount: 2, mixedImageOrderFirst: true, blackCards: 2, totalCards: 26 });
    expect(next.wordpackId).toBe('english');
    expect((next as unknown as Record<string, unknown>).unknownField).toBeUndefined();
  });

  it('parses and exports JSON5 profiles', () => {
    const profile = parseSettingsProfileJson5(`{
      name: 'Imported',
      settings: {
        imageCardCount: 2,
        mixedImageOrderFirst: true,
      },
    }`);

    expect(profile.name).toBe('Imported');
    expect(profile.settings).toMatchObject({ imageCardCount: 2, mixedImageOrderFirst: true });
    expect(exportSettingsProfileJson5(profile)).toContain("name: 'Imported'");
  });

  it('persists local profiles safely', () => {
    const storage = new MemoryStorage();
    const profiles: SettingsProfile[] = [{ id: 'custom', name: 'Custom', source: 'local', settings: { totalCards: 30 } }];
    writeSavedProfiles(storage, profiles);
    expect(storage.getItem(settingsProfilesStorageKey)).toContain('Custom');
    expect(readSavedProfiles(storage)).toEqual(profiles);

    storage.setItem(settingsProfilesStorageKey, '{broken');
    expect(readSavedProfiles(storage)).toEqual([]);
  });

  it('rejects malformed or nameless imports', () => {
    expect(() => parseSettingsProfileJson5('{broken')).toThrow(/Invalid JSON5/);
    expect(() => parseSettingsProfileJson5('{ settings: { totalCards: 30 } }')).toThrow(/name/);
  });
});
