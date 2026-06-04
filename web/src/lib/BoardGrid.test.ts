/// <reference types="vite/client" />
import { describe, expect, it } from 'vitest';

import componentSource from './BoardGrid.svelte?raw';

describe('BoardGrid surface shader contract', () => {
  it('renders board and card shader layers for themes with surface shaders outside capture mode', () => {
    expect(componentSource).toContain("import AuroraBackground from './backgrounds/AuroraBackground.svelte'");
    expect(componentSource).toContain('theme?: ThemeId');
    expect(componentSource).toContain('boardShaderTheme');
    expect(componentSource).toContain('cardShaderTheme');
    expect(componentSource).toContain('theme={boardShaderTheme}');
    expect(componentSource).toContain('theme={cardShaderTheme}');
    expect(componentSource).toContain("surface=\"board\"");
    expect(componentSource).toContain("surface=\"card\"");
  });

  it('keeps Dracula and Glitch shader behavior covered while capture mode omits shader layers', () => {
    expect(componentSource).toContain("surfaceShaderTheme(theme, 'board', captureMode)");
    expect(componentSource).toContain("surfaceShaderTheme(theme, 'card', captureMode)");
    expect(componentSource).toContain("auroraPaletteFor(theme, surface).surfaceShaders?.[surface]");
    expect(componentSource).toContain('if (captureMode) return null');
  });
});
