export const imageCardColorBorderWidthPx = 10;

export interface GameTermForms {
  one: string;
  many: string;
}

export const gameTerms = {
  card: {
    assassin: { one: 'Assassin', many: 'Assassins' },
    civilian: { one: 'Civilian', many: 'Civilians' },
    target: { one: 'Target', many: 'Targets' },
    unity: { one: 'Unity', many: 'Unity' },
  },
  role: {
    spymaster: { one: 'Spymaster', many: 'Spymasters' },
    spy: { one: 'Spy', many: 'Spies' },
    guesser: { one: 'Guesser', many: 'Guessers' },
    observer: { one: 'Observer', many: 'Observers' },
    representative: { one: 'Representative', many: 'Representatives' },
    moderator: { one: 'Moderator', many: 'Moderators' },
  },
} as const satisfies {
  card: Record<string, GameTermForms>;
  role: Record<string, GameTermForms>;
};

export function gameTermForCount(term: GameTermForms, count: number): string {
  return count === 1 ? term.one : term.many;
}

export function gameTermCountLabel(term: GameTermForms, count: number): string {
  return `${gameTermForCount(term, count)} ${count}`;
}

export function lowerGameTerm(term: string): string {
  return term.toLocaleLowerCase('en-US');
}
