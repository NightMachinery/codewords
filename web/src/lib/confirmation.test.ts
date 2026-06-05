import { describe, expect, it } from 'vitest';

import { gameTerms, lowerGameTerm } from './constants';

import {
  buildPassConfirmation,
  buildRestartConfirmation,
  buildRevealConfirmation,
  buildSwitchUnitySpymasterConfirmation,
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
      label: 'this picture card',
      imageUrl: '/api/pictures/card-1',
    });

    expect(request.cardPreview).toEqual({
      label: 'this picture card',
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

  it('builds an action-specific Unity spy switch confirmation', () => {
    const request = buildSwitchUnitySpymasterConfirmation();

    expect(request).toMatchObject({
      kind: 'switchUnitySpymaster',
      title: `Switch Unity ${lowerGameTerm(gameTerms.role.spy.one)}?`,
      message: `Move the live Unity board to the next eligible player. If this ${lowerGameTerm(gameTerms.role.spy.one)} has given a clue or revealed a card, this spends their turn.`,
      confirmLabel: `Switch ${gameTerms.role.spy.one}`,
      cancelLabel: 'Stay Here',
      tone: 'pass',
    });
  });
});
