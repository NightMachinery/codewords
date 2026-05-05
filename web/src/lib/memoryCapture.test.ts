import { describe, expect, it } from 'vitest';

import { memoryCaptureFilename, cardCaptureColors, type MemoryCaptureColorName } from './memoryCapture';

const colors: MemoryCaptureColorName[] = ['blue', 'red', 'civilian', 'black', 'hidden'];

describe('memory capture canvas helpers', () => {
  it('builds a stable safe png filename from the room id', () => {
    expect(memoryCaptureFilename('Room ABC')).toBe('codewords-memory-Room-ABC.png');
    expect(memoryCaptureFilename('../odd room!')).toBe('codewords-memory-odd-room.png');
    expect(memoryCaptureFilename('')).toBe('codewords-memory-board.png');
  });

  it('maps every board color to capture drawing colors', () => {
    for (const color of colors) {
      const mapped = cardCaptureColors(color);
      expect(mapped.fill).toMatch(/^#/);
      expect(mapped.stroke).toMatch(/^#/);
      expect(mapped.text).toMatch(/^#/);
    }
    expect(cardCaptureColors('black').fill).not.toBe(cardCaptureColors('civilian').fill);
  });
});
