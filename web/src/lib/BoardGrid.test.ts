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

  it('has theme-specific light Christmas shader overlay tuning hooks', () => {
    expect(componentSource).toContain(":global([data-theme='christmas-snow']) .surface-card-shader");
    expect(componentSource).toContain(":global([data-theme='christmas-candy']) .surface-card-shader");
  });


  it('has Blood shader card drip tuning hooks', () => {
    expect(componentSource).toContain(":global([data-theme='blood']) .surface-card-shader");
    expect(componentSource).toContain(":global([data-theme='blood']) .surface-shader-board-card");
    expect(componentSource).toContain(":global([data-theme='blood']) .surface-shader-board-card:hover:not(:disabled)");
  });


  it('resolves visible non-team card colors through centralized semantic chrome', () => {
    expect(componentSource).toContain('cardChromeColor(view.visibleColor, settings)');
    expect(componentSource).not.toContain("card.color === 'blue' ? teamColor('blue', settings)");
  });

  it('keeps Dracula revealed civilian cards distinct from unrevealed cards', () => {
    const baseRuleIndex = componentSource.indexOf(":global([data-theme='dracula']) .surface-shader-board-card {");
    const civilianRuleIndex = componentSource.indexOf(":global([data-theme='dracula']) .surface-shader-board-card.revealed-civilian-card");

    expect(baseRuleIndex).toBeGreaterThan(-1);
    expect(civilianRuleIndex).toBeGreaterThan(baseRuleIndex);
    expect(componentSource).toContain('background-color: oklch(31% 0.075 76 / 0.9)');
    expect(componentSource).toContain('border-color: oklch(83% 0.13 82 / 0.72)');
  });

  it('passes number-badge avoidance only into fitted word labels', () => {
    expect(componentSource).toContain('cardWordAvoidsTopLeftBadge');
    expect(componentSource).toContain('avoidTopLeftBadge={preferences.showNumberBadges && cardWordAvoidsTopLeftBadge(card.word)}');
    expect(componentSource).toContain('<FitCardWord');

    const imageBranchStart = componentSource.indexOf("{#if card.contentType === 'image'}");
    const wordBranchStart = componentSource.indexOf('<FitCardWord');
    expect(imageBranchStart).toBeGreaterThan(-1);
    expect(wordBranchStart).toBeGreaterThan(imageBranchStart);
  });
});
