import JSON5 from 'json5';

import type { Settings } from './api';
import { normalizeLobbySettingsForSave } from './gameplay';

export const settingsProfilesStorageKey = 'codewords.settingsProfiles';

export type SettingsProfileSource = 'bundled' | 'local';

export interface SettingsProfile {
  id: string;
  name: string;
  source: SettingsProfileSource;
  settings: Partial<Settings>;
}

const settingKeys = new Set<keyof Settings>([
  'seed',
  'blackCards',
  'totalCards',
  'autoColorCounts',
  'blueCards',
  'redCards',
  'neutralCards',
  'wordpackId',
  'enforceClueGuessLimit',
  'allowInfinityClue',
  'imageCardCount',
  'randomizeTeams',
  'customColorBlue',
  'customColorRed',
  'teamNameBlue',
  'teamNameRed',
  'observerChatEnabled',
  'mixedImageOrderFirst',
]);

export function applySettingsProfile(current: Settings, profile: Pick<SettingsProfile, 'settings'>): Settings {
  const known: Partial<Settings> = {};
  const raw = profile.settings as Record<string, unknown>;
  for (const key of settingKeys) {
    if (Object.prototype.hasOwnProperty.call(raw, key)) {
      (known as Record<string, unknown>)[key] = raw[key];
    }
  }
  return normalizeLobbySettingsForSave({ ...current, ...known });
}

export function parseSettingsProfileJson5(text: string): SettingsProfile {
  let parsed: unknown;
  try {
    parsed = JSON5.parse(text);
  } catch (error) {
    throw new Error(`Invalid JSON5 profile: ${error instanceof Error ? error.message : 'parse failed'}`);
  }
  if (!parsed || typeof parsed !== 'object') throw new Error('Settings profile must be an object.');
  const record = parsed as Record<string, unknown>;
  if (typeof record.name !== 'string' || !record.name.trim()) throw new Error('Settings profile needs a name.');
  if (!record.settings || typeof record.settings !== 'object') throw new Error('Settings profile needs a settings object.');
  return {
    id: typeof record.id === 'string' && record.id.trim() ? record.id.trim() : slugify(record.name),
    name: record.name.trim(),
    source: record.source === 'bundled' ? 'bundled' : 'local',
    settings: knownSettings(record.settings as Record<string, unknown>),
  };
}

export function exportSettingsProfileJson5(profile: SettingsProfile): string {
  return JSON5.stringify({ name: profile.name, settings: knownSettings(profile.settings as Record<string, unknown>) }, null, 2);
}

export function readSavedProfiles(storage: Pick<Storage, 'getItem'>): SettingsProfile[] {
  const raw = storage.getItem(settingsProfilesStorageKey);
  if (!raw) return [];
  try {
    const parsed = JSON.parse(raw) as unknown;
    if (!Array.isArray(parsed)) return [];
    return parsed.flatMap((item) => {
      if (!item || typeof item !== 'object') return [];
      try {
        const record = item as Record<string, unknown>;
        if (typeof record.name !== 'string' || !record.settings || typeof record.settings !== 'object') return [];
        return [{ id: typeof record.id === 'string' ? record.id : slugify(record.name), name: record.name, source: 'local' as const, settings: knownSettings(record.settings as Record<string, unknown>) }];
      } catch {
        return [];
      }
    });
  } catch {
    return [];
  }
}

export function writeSavedProfiles(storage: Pick<Storage, 'setItem'>, profiles: SettingsProfile[]): void {
  const localProfiles = profiles.map((profile) => ({ ...profile, source: 'local' as const, settings: knownSettings(profile.settings as Record<string, unknown>) }));
  storage.setItem(settingsProfilesStorageKey, JSON.stringify(localProfiles));
}

export function profileFromSettings(name: string, settings: Settings): SettingsProfile {
  return { id: `${slugify(name)}-${Date.now().toString(36)}`, name: name.trim(), source: 'local', settings: knownSettings(settings as unknown as Record<string, unknown>) };
}

function knownSettings(raw: Record<string, unknown>): Partial<Settings> {
  const known: Record<string, unknown> = {};
  for (const key of settingKeys) {
    if (Object.prototype.hasOwnProperty.call(raw, key)) known[key] = raw[key];
  }
  return known as Partial<Settings>;
}

function slugify(value: string): string {
  return value.trim().toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/(^-|-$)/g, '') || 'profile';
}
