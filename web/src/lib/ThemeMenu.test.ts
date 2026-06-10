/// <reference types="vite/client" />
import { describe, expect, it } from 'vitest';

import source from './ThemeMenu.svelte?raw';

describe('ThemeMenu mobile layout contract', () => {
  it('constrains the popover to the viewport and scrolls long theme lists', () => {
    expect(source).toContain('w-[calc(100vw-1.5rem)]');
    expect(source).toContain('max-w-72');
    expect(source).toContain('max-h-[min(calc(100dvh-5rem),32rem)]');
    expect(source).toContain('overflow-y-auto');
  });

  it('keeps theme labels and controls inside narrow popovers', () => {
    expect(source).toContain('min-w-0');
    expect(source).toContain('truncate');
    expect(source).toContain('max-w-full');
  });
});
