import { describe, expect, it } from 'vitest';

import { appMeta } from './appMeta';

describe('appMeta', () => {
  it('identifies the scaffolded app without external asset dependencies', () => {
    expect(appMeta).toEqual({
      name: 'Codewords',
      tagline: 'Self-hosted word and picture deduction for your table',
      externalAssets: false,
    });
  });
});
