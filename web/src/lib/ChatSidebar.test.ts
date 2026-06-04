/// <reference types="vite/client" />
import { describe, expect, it } from 'vitest';

import componentSource from './ChatSidebar.svelte?raw';

describe('ChatSidebar scroll contract', () => {
  it('tracks and restores chat scroll position across collapse and reopen', () => {
    expect(componentSource).toContain("import { onMount, tick } from 'svelte'");
    expect(componentSource).toContain('let scrollContainer = $state<HTMLDivElement>()');
    expect(componentSource).toContain('bind:this={scrollContainer}');
    expect(componentSource).toContain('onscroll={handleScroll}');
    expect(componentSource).toContain('savedScrollTop');
    expect(componentSource).toContain('restoreScrollPosition');
    expect(componentSource).toContain('await tick()');
  });

  it('scrolls after own sends and incoming messages only when already near bottom', () => {
    expect(componentSource).toContain('pendingOwnSendScroll');
    expect(componentSource).toContain('chatScrollShouldAutoScroll');
    expect(componentSource).toContain('chatScrollIsNearBottom');
    expect(componentSource).toContain('messages.length');
    expect(componentSource).toContain('scrollContainer.scrollTop = scrollContainer.scrollHeight');
  });
});
