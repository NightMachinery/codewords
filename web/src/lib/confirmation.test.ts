import { describe, expect, it } from 'vitest';

import {
  buildPassConfirmation,
  buildRestartConfirmation,
  buildRevealConfirmation,
} from './confirmation';

describe('confirmation request builders', () => {
  it('builds an action-specific reveal confirmation with the card label', () => {
    const request = buildRevealConfirmation({ label: 'Moon Base' });

    expect(request).toMatchObject({
      kind: 'reveal',
      title: 'Reveal card?',
      message: 'Reveal Moon Base to the room.',
      confirmLabel: 'Reveal',
      cancelLabel: 'Keep Hidden',
      tone: 'reveal',
    });
  });

  it('includes an image-card preview for picture reveal confirmations', () => {
    const request = buildRevealConfirmation({
      label: 'Picture card',
      imageUrl: '/api/pictures/card-1',
    });

    expect(request.cardPreview).toEqual({
      label: 'Picture card',
      imageUrl: '/api/pictures/card-1',
    });
  });

  it('builds an action-specific pass confirmation', () => {
    const request = buildPassConfirmation();

    expect(request).toMatchObject({
      kind: 'pass',
      title: 'Pass turn?',
      message: 'End guessing for this turn and hand play to the next team.',
      confirmLabel: 'Pass',
      cancelLabel: 'Keep Guessing',
      tone: 'pass',
    });
  });

  it('builds an action-specific restart confirmation', () => {
    const request = buildRestartConfirmation();

    expect(request).toMatchObject({
      kind: 'restart',
      title: 'Restart match?',
      message: 'Return everyone to the lobby and generate a fresh board for the next start.',
      confirmLabel: 'Restart',
      cancelLabel: 'Stay Here',
      tone: 'danger',
    });
  });
});
