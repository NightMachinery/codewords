import { describe, expect, it } from 'vitest';

import { svgAssetMarkup } from './customSvg';

describe('custom SVG asset rendering', () => {
  it('renders trusted SVG asset markup without CSS mask square fallback', () => {
    const markup = svgAssetMarkup('<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill="currentColor" d="M1 1h2v2H1z"/></svg>', 'Spy', 'h-4 w-4');

    expect(markup).toContain('<svg');
    expect(markup).toContain('class="h-4 w-4"');
    expect(markup).toContain('aria-label="Spy"');
    expect(markup).not.toContain('mask:');
    expect(markup).not.toContain('bg-current');
  });
});
