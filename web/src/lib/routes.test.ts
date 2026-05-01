import { describe, expect, it } from 'vitest';

import { roomPath, websocketRoomUrl } from './routes';

describe('route helpers', () => {
  it('builds canonical room paths', () => {
    expect(roomPath('abc123')).toBe('/rooms/abc123');
  });

  it('uses ws for http pages', () => {
    expect(websocketRoomUrl(new URL('http://lan.test/rooms/abc'), 'abc', { authToken: 'token one' })).toBe(
      'ws://lan.test/ws/rooms/abc?authToken=token+one',
    );
  });

  it('uses wss for https pages and migrate ids', () => {
    expect(websocketRoomUrl(new URL('https://play.test/rooms/abc'), 'abc', { migrateId: 'mig' })).toBe(
      'wss://play.test/ws/rooms/abc?migrateId=mig',
    );
  });
});
