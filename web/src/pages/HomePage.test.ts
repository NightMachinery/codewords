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

  it('renders the reusable WebGL aurora background behind the hero surface', () => {
    expect(source).toContain("import AuroraBackground from '../lib/backgrounds/AuroraBackground.svelte'");
    expect(source).toContain('<AuroraBackground');
    expect(source).toContain('hero-shell');
    expect(source).not.toContain('aurora-ribbon');
    expect(source).not.toContain('@keyframes aurora-drift');
  });
});
