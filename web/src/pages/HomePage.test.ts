/// <reference types="vite/client" />
import { describe, expect, it } from 'vitest';

import source from './HomePage.svelte?raw';

describe('HomePage minimal landing surface', () => {
  it('removes marketing copy from the landing page', () => {
    expect(source).not.toContain('Private team wordplay');
    expect(source).not.toContain('Start a Codewords table in seconds.');
    expect(source).not.toContain('Self-hosted rooms, local assets, no accounts.');
    expect(source).not.toContain('Create a room, share the link');
  });

  it('uses an animated aurora layer with reduced-motion handling', () => {
    expect(source).toContain('aurora-background');
    expect(source).toContain('aurora-ribbon');
    expect(source).toContain('@keyframes aurora-drift');
    expect(source).toContain('prefers-reduced-motion: reduce');
  });
});
