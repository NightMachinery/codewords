/// <reference types="vite/client" />
import { describe, expect, it } from 'vitest';

import source from './RoomPage.svelte?raw';

describe('RoomPage confirmation contract', () => {
  it('uses the shared styled confirmation popup instead of native browser confirms', () => {
    expect(source).not.toContain('window.confirm');
    expect(source).toContain("import ConfirmPopup from '../lib/ConfirmPopup.svelte'");
    expect(source).toContain('<ConfirmPopup');
    expect(source).toContain('buildRevealConfirmation');
    expect(source).toContain('buildPassConfirmation');
    expect(source).toContain('buildRestartConfirmation');
  });
});
