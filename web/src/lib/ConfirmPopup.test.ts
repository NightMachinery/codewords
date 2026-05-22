/// <reference types="vite/client" />
import { describe, expect, it } from 'vitest';

import source from './ConfirmPopup.svelte?raw';

describe('ConfirmPopup component contract', () => {
  it('renders an accessible reusable confirmation dialog', () => {
    expect(source).toContain('role="dialog"');
    expect(source).toContain('aria-modal="true"');
    expect(source).toContain('onkeydown');
    expect(source).toContain("event.key === 'Escape'");
    expect(source).toContain('request.tone');
    expect(source).toContain('request.cardPreview');
    expect(source).toContain('src={request.cardPreview.imageUrl}');
    expect(source).toContain('onclick={onConfirm}');
    expect(source).toContain('onclick={onCancel}');
  });
});
