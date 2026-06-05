import { describe, expect, it } from 'vitest';

import { gameTermCountLabel, gameTerms } from './constants';

describe('game terminology constants', () => {
  it('centralizes default user-facing card and role labels', () => {
    expect(gameTerms.card.assassin.one).toBe('Assassin');
    expect(gameTerms.card.assassin.many).toBe('Assassins');
    expect(gameTerms.card.civilian.one).toBe('Civilian');
    expect(gameTerms.card.civilian.many).toBe('Civilians');
    expect(gameTerms.role.spymaster.one).toBe('Spymaster');
    expect(gameTerms.role.spymaster.many).toBe('Spymasters');
    expect(gameTerms.role.spy.one).toBe('Spy');
    expect(gameTerms.role.guesser.one).toBe('Guesser');
    expect(gameTerms.role.observer.many).toBe('Observers');
    expect(gameTerms.role.representative.one).toBe('Representative');
  });

  it('formats accessible count labels from shared terms', () => {
    expect(gameTermCountLabel(gameTerms.card.assassin, 1)).toBe('Assassin 1');
    expect(gameTermCountLabel(gameTerms.card.assassin, 2)).toBe('Assassins 2');
    expect(gameTermCountLabel(gameTerms.card.civilian, 0)).toBe('Civilians 0');
  });
});
