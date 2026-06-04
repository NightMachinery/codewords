/// <reference types="vite/client" />
import { describe, expect, it } from 'vitest';

import componentSource from './BoardGrid.svelte?raw';

describe('BoardGrid Dracula shader contract', () => {
  it('renders board and card shader layers for Dracula outside capture mode', () => {
    expect(componentSource).toContain("import AuroraBackground from './backgrounds/AuroraBackground.svelte'");
    expect(componentSource).toContain('theme?: ThemeId');
    expect(componentSource).toContain("theme === 'dracula' && !captureMode");
    expect(componentSource).toContain('surface="board"');
    expect(componentSource).toContain('surface="card"');
  });
});
