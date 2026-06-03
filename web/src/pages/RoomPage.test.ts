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

  it('keeps the Power restart icon visible on light themes with local styling', () => {
    expect(source).toContain("import Power from 'lucide-svelte/icons/power'");
    expect(source).toContain('<Power class="h-4 w-4" />');
    expect(source).toContain('restart-match-button');
    expect(source).toContain(":global([data-theme='light']) .restart-match-button");
    expect(source).toContain(":global([data-theme='solarized-light']) .restart-match-button");
  });
});
