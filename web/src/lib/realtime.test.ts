import { describe, expect, it, vi } from 'vitest';

import { RoomSocket } from './realtime';

class FakeWebSocket {
  static instances: FakeWebSocket[] = [];
  static OPEN = 1;

  readyState = 0;
  sent: string[] = [];
  onopen: (() => void) | null = null;
  onmessage: ((event: { data: string }) => void) | null = null;
  onclose: (() => void) | null = null;
  onerror: (() => void) | null = null;

  constructor(readonly url: string) {
    FakeWebSocket.instances.push(this);
  }

  send(message: string): void {
    this.sent.push(message);
  }

  close(): void {
    this.onclose?.();
  }
}

describe('RoomSocket', () => {
  it('reports whether an action was sent over an open socket', () => {
    const original = globalThis.WebSocket;
    vi.stubGlobal('WebSocket', FakeWebSocket);
    try {
      const socket = new RoomSocket('ws://room', { onMessage: () => {}, onStatus: () => {} });
      socket.connect();
      const instance = FakeWebSocket.instances.at(-1)!;

      expect(socket.send({ type: 'guessCard', index: 1 })).toBe(false);
      expect(instance.sent).toEqual([]);

      instance.readyState = FakeWebSocket.OPEN;

      expect(socket.send({ type: 'guessCard', index: 1 })).toBe(true);
      expect(instance.sent).toEqual([JSON.stringify({ type: 'guessCard', index: 1 })]);
    } finally {
      vi.stubGlobal('WebSocket', original);
    }
  });
});
