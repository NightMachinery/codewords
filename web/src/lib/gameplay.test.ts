import { describe, expect, it } from 'vitest';

import {
  canSubmitClue,
  cardViewState,
  defaultGameplayPreferences,
  formatClueNumber,
  isActiveGuesser,
  parseClueNumber,
  readGameplayPreferences,
  shouldAutoJoinRoom,
  viewerRole,
  writeGameplayPreferences,
  type GameplayCard,
  type GameplayPreferences,
} from './gameplay';
import type { RoomSummary, Settings, Viewer } from './api';
import type { LobbyPlayer } from './lobby';

class MemoryStorage {
  values = new Map<string, string>();
  getItem(key: string): string | null {
    return this.values.get(key) ?? null;
  }
  setItem(key: string, value: string): void {
    this.values.set(key, value);
  }
}

const players: LobbyPlayer[] = [
  { id: 'blueSpy', displayName: 'Blue Spy', team: 'blue', spymaster: true, representative: false },
  { id: 'blueGuess', displayName: 'Blue Guess', team: 'blue', spymaster: false, representative: false },
  { id: 'redSpy', displayName: 'Red Spy', team: 'red', spymaster: true, representative: false },
  { id: 'redGuess', displayName: 'Red Guess', team: 'red', spymaster: false, representative: false },
];
const settings: Settings = { seed: 1, blackCards: 1, wordpackId: 'english', enforceClueGuessLimit: false, allowInfinityClue: false };

function viewer(userId: string): Viewer {
  return { userId, playerId: userId, isHost: false };
}

describe('gameplay permissions', () => {
  it('mirrors backend active guesser rules for regular, representative, spectator, and all-spymaster teams', () => {
    expect(isActiveGuesser(players, 'blueGuess', 'blue')).toBe(true);
    expect(isActiveGuesser(players, 'blueSpy', 'blue')).toBe(false);
    expect(isActiveGuesser(players, 'redGuess', 'blue')).toBe(false);
    expect(isActiveGuesser(players, 'spectator', 'blue')).toBe(false);

    const representativePlayers = players.map((player) =>
      player.id === 'blueGuess' ? { ...player, representative: true } : player,
    );
    expect(isActiveGuesser(representativePlayers, 'blueGuess', 'blue')).toBe(true);
    expect(isActiveGuesser(representativePlayers, 'blueSpy', 'blue')).toBe(false);

    const twoSpies: LobbyPlayer[] = [
      { id: 'spy1', displayName: 'Spy 1', team: 'blue', spymaster: true, representative: false },
      { id: 'spy2', displayName: 'Spy 2', team: 'blue', spymaster: true, representative: false },
    ];
    expect(isActiveGuesser(twoSpies, 'spy1', 'blue')).toBe(true);
    expect(isActiveGuesser(twoSpies, 'spy2', 'blue')).toBe(true);
  });

  it('identifies spectator, player, spymaster, and active guesser view roles', () => {
    expect(viewerRole(players, viewer('spectator'), 'blue')).toMatchObject({ kind: 'spectator', canSeeHiddenColors: false, activeGuesser: false });
    expect(viewerRole(players, viewer('blueGuess'), 'blue')).toMatchObject({ kind: 'player', team: 'blue', canSeeHiddenColors: false, activeGuesser: true });
    expect(viewerRole(players, viewer('blueSpy'), 'blue')).toMatchObject({ kind: 'spymaster', team: 'blue', canSeeHiddenColors: true, activeGuesser: false });
    expect(viewerRole(players, viewer('blueSpy'), 'blue', 'game_over')).toMatchObject({ canSeeHiddenColors: true });
    expect(viewerRole(players, viewer('blueGuess'), 'blue', 'game_over')).toMatchObject({ canSeeHiddenColors: true, activeGuesser: false });
  });

  it('allows only the current team spymaster to submit clues during active play', () => {
    expect(canSubmitClue(players, viewer('blueSpy'), 'blue', 'active').allowed).toBe(true);
    expect(canSubmitClue(players, viewer('redSpy'), 'blue', 'active').reason).toBe('Only the blue spymaster can clue right now.');
    expect(canSubmitClue(players, viewer('blueGuess'), 'blue', 'active').reason).toBe('Only spymasters can clue.');
    expect(canSubmitClue(players, viewer('spectator'), 'blue', 'active').reason).toBe('Spectators are read-only.');
    expect(canSubmitClue(players, viewer('blueSpy'), 'blue', 'game_over').reason).toBe('The match is over.');
  });
});

describe('clue number helpers', () => {
  it('parses and formats blank, numeric, and infinity clue numbers', () => {
    expect(parseClueNumber('')).toEqual({ kind: 'blank' });
    expect(parseClueNumber('4')).toEqual({ kind: 'numeric', value: 4 });
    expect(parseClueNumber('12')).toEqual({ kind: 'blank' });
    expect(parseClueNumber('∞')).toEqual({ kind: 'infinity' });
    expect(formatClueNumber({ kind: 'blank' })).toBe('any');
    expect(formatClueNumber({ kind: 'numeric', value: 7 })).toBe('7');
    expect(formatClueNumber({ kind: 'infinity' })).toBe('∞');
  });
});

describe('local gameplay preferences', () => {
  it('defaults confirmation preferences and persists partial updates', () => {
    const storage = new MemoryStorage();
    expect(readGameplayPreferences(storage)).toEqual(defaultGameplayPreferences);

    const saved: GameplayPreferences = { confirmGuesses: false, confirmPasses: true };
    writeGameplayPreferences(storage, saved);
    expect(readGameplayPreferences(storage)).toEqual(saved);

    storage.setItem('codewords.gameplayPreferences', JSON.stringify({ confirmGuesses: false }));
    expect(readGameplayPreferences(storage)).toEqual({ confirmGuesses: false, confirmPasses: defaultGameplayPreferences.confirmPasses });
  });
});

describe('board card state', () => {
  const hiddenBlue: GameplayCard = { contentType: 'word', word: 'river', revealed: false, color: 'blue' };
  const revealedRed: GameplayCard = { contentType: 'word', word: 'castle', revealed: true, color: 'red' };
  const hiddenUnknown: GameplayCard = { contentType: 'word', word: 'orbit', revealed: false };

  it('derives visible color, labels, classes, and last-selected state', () => {
    expect(cardViewState(hiddenUnknown, 0, false, undefined)).toMatchObject({ visibleColor: 'hidden', label: 'Unrevealed', isLastSelected: false });
    expect(cardViewState(hiddenBlue, 1, true, undefined)).toMatchObject({ visibleColor: 'blue', label: 'Blue', isLastSelected: false });
    expect(cardViewState(revealedRed, 2, false, { index: 2, team: 'red' })).toMatchObject({ visibleColor: 'red', label: 'Red', isLastSelected: true });
    expect(cardViewState(revealedRed, 2, false, { index: 2, team: 'red' }).classes).toContain('ring-4');
  });
});

describe('active-room boot behavior', () => {
  it('auto-joins named auth users only while the room is still a lobby', () => {
    const lobbyRoom: RoomSummary = { id: 'room', hostUserId: 'host', status: 'lobby', currentMatchId: '' };
    const activeRoom: RoomSummary = { ...lobbyRoom, status: 'active', currentMatchId: 'match' };

    expect(shouldAutoJoinRoom(lobbyRoom, 'auth', 'Alice')).toBe(true);
    expect(shouldAutoJoinRoom(activeRoom, 'auth', 'Alice')).toBe(false);
    expect(shouldAutoJoinRoom(lobbyRoom, 'auth', '')).toBe(false);
    expect(shouldAutoJoinRoom(lobbyRoom, 'migrate', 'Alice')).toBe(false);
  });
});
