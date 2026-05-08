import { describe, expect, it } from 'vitest';

import { downloadMemoryCapture, memoryCaptureFilename, cardCaptureColors, memoryCaptureDesktopWidth, waitForMemoryCaptureReady, type MemoryCaptureColorName } from './memoryCapture';

const colors: MemoryCaptureColorName[] = ['blue', 'red', 'civilian', 'black', 'hidden'];

describe('memory capture canvas helpers', () => {
  it('builds a stable safe png filename from the room id', () => {
    expect(memoryCaptureFilename('Room ABC')).toBe('codewords-memory-Room-ABC.png');
    expect(memoryCaptureFilename('../odd room!')).toBe('codewords-memory-odd-room.png');
    expect(memoryCaptureFilename('')).toBe('codewords-memory-board.png');
  });

  it('exports a provided board DOM node as the memory PNG', async () => {
    const clicks: string[] = [];
    const urls: string[] = [];
    const blob = new Blob(['png'], { type: 'image/png' });
    const node = { dataset: { capture: 'board' } } as unknown as HTMLElement;
    const body = { appendChild: (link: { download: string }) => { clicks.push(link.download); } };
    const doc = {
      body,
      createElement: () => ({
        href: '',
        download: '',
        rel: '',
        click: () => clicks.push('clicked'),
        remove: () => clicks.push('removed'),
      }),
    } as unknown as Document;
    const previousUrl = globalThis.URL;
    Object.defineProperty(globalThis, 'URL', {
      configurable: true,
      value: {
        createObjectURL: (value: Blob) => {
          expect(value).toBe(blob);
          urls.push('created');
          return 'blob:memory';
        },
        revokeObjectURL: (url: string) => urls.push(`revoked:${url}`),
      },
    });

    try {
      await downloadMemoryCapture({ roomId: 'Room ABC' }, node, doc, async (exportNode) => {
        expect(exportNode).toBe(node);
        return blob;
      });
    } finally {
      Object.defineProperty(globalThis, 'URL', { configurable: true, value: previousUrl });
    }

    expect(clicks).toEqual(['codewords-memory-Room-ABC.png', 'clicked', 'removed']);
    expect(urls).toEqual(['created', 'revoked:blob:memory']);
  });

  it('uses a fixed desktop export width for memory captures', () => {
    expect(memoryCaptureDesktopWidth).toBe(1400);
  });

  it('waits for fonts, images, and fitted word labels before rendering', async () => {
    const events: string[] = [];
    let fitReady = 'false';
    const fitNode = {
      getAttribute: (name: string) => name === 'data-fit-ready' ? fitReady : null,
    };
    const image = {
      complete: false,
      decode: async () => {
        events.push('image decoded');
      },
    };
    const node = {
      ownerDocument: {
        fonts: {
          ready: Promise.resolve().then(() => {
            events.push('fonts ready');
          }),
        },
      },
      querySelectorAll: (selector: string) => {
        if (selector === '[data-fit-card-word]') return [fitNode];
        if (selector === 'img') return [image];
        return [];
      },
    } as unknown as HTMLElement;
    const win = {
      requestAnimationFrame: (callback: FrameRequestCallback) => {
        events.push('animation frame');
        fitReady = 'true';
        callback(0);
        return 1;
      },
    } as unknown as Window;

    await waitForMemoryCaptureReady(node, win);

    expect(events).toEqual(['fonts ready', 'image decoded', 'animation frame']);
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
