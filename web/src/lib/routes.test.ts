import { describe, expect, it } from 'vitest';

import { canonicalHost, canonicalOrigin, roomPath, websocketRoomUrl } from './routes';

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

  it('strips trailing DNS root dots from generated origins and websocket hosts', () => {
    expect(canonicalHost('codewords.pinky.lilf.ir.')).toBe('codewords.pinky.lilf.ir');
    expect(canonicalOrigin(new URL('https://codewords.pinky.lilf.ir./rooms/abc'))).toBe('https://codewords.pinky.lilf.ir');
    expect(websocketRoomUrl(new URL('https://codewords.pinky.lilf.ir./rooms/abc'), 'abc', { authToken: 'token' })).toBe(
      'wss://codewords.pinky.lilf.ir/ws/rooms/abc?authToken=token',
    );
  });
});
