import { describe, expect, it } from 'vitest';

import {
  authTokenStorageKey,
  getOrCreateAuthToken,
  resolveSessionCredential,
  type BrowserStorage,
} from './identity';

class MemoryStorage implements BrowserStorage {
  private values = new Map<string, string>();

  getItem(key: string): string | null {
    return this.values.get(key) ?? null;
  }

  setItem(key: string, value: string): void {
    this.values.set(key, value);
  }
}

describe('identity helpers', () => {
  it('reuses a saved browser auth token', () => {
    const storage = new MemoryStorage();
    storage.setItem(authTokenStorageKey, 'saved-token');

    expect(getOrCreateAuthToken(storage, () => 'new-token')).toBe('saved-token');
    expect(storage.getItem(authTokenStorageKey)).toBe('saved-token');
  });

  it('creates and persists a browser auth token when none exists', () => {
    const storage = new MemoryStorage();

    expect(getOrCreateAuthToken(storage, () => 'created-token')).toBe('created-token');
    expect(storage.getItem(authTokenStorageKey)).toBe('created-token');
  });

  it('uses a room migrate id without overwriting the global auth token', () => {
    const storage = new MemoryStorage();
    storage.setItem(authTokenStorageKey, 'global-token');
    const url = new URL('http://example.test/rooms/room-1?migrateId=room-only');

    expect(resolveSessionCredential(url, storage, () => 'new-token')).toEqual({
      mode: 'migrate',
      migrateId: 'room-only',
      authToken: 'global-token',
    });
    expect(storage.getItem(authTokenStorageKey)).toBe('global-token');
  });
});
