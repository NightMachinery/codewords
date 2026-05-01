import { describe, expect, it } from 'vitest';

import { copyText } from './clipboard';

describe('copyText', () => {
  it('uses navigator.clipboard when available', async () => {
    const writes: string[] = [];

    const result = await copyText('room-link', {
      navigator: {
        clipboard: {
          writeText: async (value: string) => {
            writes.push(value);
          },
        },
      },
    });

    expect(result).toEqual({ ok: true, method: 'clipboard' });
    expect(writes).toEqual(['room-link']);
  });

  it('falls back to a temporary textarea when async clipboard is unavailable', async () => {
    const selected: string[] = [];
    const children: unknown[] = [];

    const textarea = {
      value: '',
      style: { position: '', left: '', top: '' },
      focus: () => undefined,
      select: () => selected.push(textarea.value),
    };
    const document = {
      body: {
        appendChild: (node: unknown) => children.push(node),
        removeChild: (node: unknown) => children.splice(children.indexOf(node), 1),
      },
      createElement: () => textarea,
      execCommand: (command: string) => command === 'copy',
    };

    const result = await copyText('fallback-link', { document });

    expect(result).toEqual({ ok: true, method: 'execCommand' });
    expect(selected).toEqual(['fallback-link']);
    expect(children).toEqual([]);
  });

  it('reports manual copy when all copy methods fail', async () => {
    const result = await copyText('raw-link', {});

    expect(result).toEqual({ ok: false, method: 'manual' });
  });
});
