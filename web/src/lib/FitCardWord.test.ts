/// <reference types="vite/client" />
import { describe, expect, it } from 'vitest';

import source from './FitCardWord.svelte?raw';

describe('FitCardWord badge avoidance contract', () => {
  it('keeps the label centered and clears the badge by symmetric shrinking, not floating', () => {
    expect(source).toContain('avoidTopLeftBadge?: boolean');
    // The label is a plain centered span — no float placeholder that would drag
    // it off-center vertically.
    expect(source).not.toContain('float: left');
    expect(source).not.toContain('fit-card-word--avoid-top-left-badge');
    expect(source).not.toContain('centeredLabelAvoidsTopLeftBadge');
  });

  it('reserves the badge band symmetrically only when the centered label enters it', () => {
    // Roomy cards fit at full size; a colliding label is re-fitted into a
    // height with the badge band reserved on both ends so it stays centered.
    expect(source).toContain('centredLabelEntersBadgeBand');
    expect(source).toContain('avoidTopLeftBadge && centredLabelEntersBadgeBand(');
    expect(source).toContain('height - 2 * badgeBandPx');
  });
});
