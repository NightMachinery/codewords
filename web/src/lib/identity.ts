export const authTokenStorageKey = 'codewords.authToken';

export interface BrowserStorage {
  getItem(key: string): string | null;
  setItem(key: string, value: string): void;
}

export type SessionCredential =
  | { mode: 'auth'; authToken: string; migrateId?: never }
  | { mode: 'migrate'; migrateId: string; authToken: string };

export function randomToken(): string {
  const bytes = new Uint8Array(32);
  crypto.getRandomValues(bytes);
  return Array.from(bytes, (byte) => byte.toString(16).padStart(2, '0')).join('');
}

export function getOrCreateAuthToken(storage: BrowserStorage, createToken = randomToken): string {
  const existing = storage.getItem(authTokenStorageKey);
  if (existing) {
    return existing;
  }
  const token = createToken();
  storage.setItem(authTokenStorageKey, token);
  return token;
}

export function resolveSessionCredential(
  url: URL,
  storage: BrowserStorage,
  createToken = randomToken,
): SessionCredential {
  const authToken = getOrCreateAuthToken(storage, createToken);
  const migrateId = url.searchParams.get('migrateId')?.trim();
  if (migrateId) {
    return { mode: 'migrate', migrateId, authToken };
  }
  return { mode: 'auth', authToken };
}
