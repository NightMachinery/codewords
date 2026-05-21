import { describe, expect, it } from 'vitest';

import {
  defaultSettings,
  type Settings,
} from './api';
import {
  defaultTeamNames,
  displayTeamName,
  isActiveGuesser,
  normalizeLobbySettingsForSave,
  teamColor,
  unityBoardViewCards,
  unityEndGameSummary,
  unityStartReadiness,
  type GameplayCard,
  type UnityBoardSnapshot,
  type UnityProgress,
} from './gameplay';
import type { LobbyPlayer } from './lobby';

const unitySettings: Settings = {
  ...defaultSettings,
  mode: 'unity',
  blackCards: 4,
  unityTurnLimit: 5,
};

const players: LobbyPlayer[] = [
  { id: 'host', displayName: 'Host', team: 'unity', spymaster: false, representative: false, mod: true },
  { id: 'p2', displayName: 'P2', team: 'unity', spymaster: false, representative: false, mod: false },
  { id: 'p3', displayName: 'P3', team: 'unity', spymaster: false, representative: false, mod: false },
];

describe('unity frontend helpers', () => {
  it('normalizes unity defaults without polarity card-count math', () => {
    const settings = normalizeLobbySettingsForSave({ ...defaultSettings, mode: 'unity', totalCards: 25, blackCards: 0, unityTurnLimit: 0 });

    expect(settings.mode).toBe('unity');
    expect(settings.blackCards).toBe(4);
    expect(settings.unityTurnLimit).toBe(9);
    expect(settings.neutralCards).toBe(15);
    expect(settings.blueCards).toBe(0);
    expect(settings.redCards).toBe(0);
    expect(defaultTeamNames.unity).toBe('Unity');
    expect(displayTeamName('unity', settings)).toBe('Unity');
    expect(teamColor('unity', settings)).toBe('#20b2aa');
  });

  it('checks unity start readiness from unity players only', () => {
    expect(unityStartReadiness(players).ready).toBe(true);
    expect(unityStartReadiness(players.slice(0, 1)).reason).toBe('Unity needs at least two active players.');
    expect(unityStartReadiness([{ ...players[0], team: '' }]).reason).toBe('Assign every player to Unity or observer mode first.');
  });

  it('mirrors unity representative guesser rules', () => {
    expect(isActiveGuesser(players, 'p2', 'unity', 'host')).toBe(true);
    expect(isActiveGuesser(players, 'host', 'unity', 'host')).toBe(false);

    const oneRep = players.map((player) => player.id === 'p2' ? { ...player, representative: true } : player);
    expect(isActiveGuesser(oneRep, 'p2', 'unity', 'host')).toBe(true);
    expect(isActiveGuesser(oneRep, 'p3', 'unity', 'host')).toBe(false);

    const ownerRep = players.map((player) => player.id === 'host' ? { ...player, representative: true } : player);
    expect(isActiveGuesser(ownerRep, 'p2', 'unity', 'host')).toBe(true);
    expect(isActiveGuesser(ownerRep, 'p3', 'unity', 'host')).toBe(true);

    const twoReps = ownerRep.map((player) => player.id === 'p2' ? { ...player, representative: true } : player);
    expect(isActiveGuesser(twoReps, 'p2', 'unity', 'host')).toBe(true);
    expect(isActiveGuesser(twoReps, 'p3', 'unity', 'host')).toBe(false);
  });

  it('selects active or own unity board cards for display', () => {
    const active: UnityBoardSnapshot = { ownerId: 'host', cards: [{ contentType: 'word', word: 'Active', revealed: false }], clueLog: [], turnsUsed: 1, remainingCounts: { unity: 9, civilian: 10, black: 4 } };
    const own: UnityBoardSnapshot = { ownerId: 'p2', cards: [{ contentType: 'word', word: 'Own', revealed: false, color: 'unity' }], clueLog: [], turnsUsed: 0, remainingCounts: { unity: 10, civilian: 11, black: 4 } };

    expect(unityBoardViewCards('active', active, own)[0].word).toBe('Active');
    expect(unityBoardViewCards('own', active, own)[0].word).toBe('Own');
    expect(unityBoardViewCards('own', active, null)[0].word).toBe('Active');
  });

  it('formats unity end game stats and shared pool progress', () => {
    const progress: UnityProgress = { unityCardsFound: 17, totalUnityCards: 30, sharedTurnsRemaining: 3, unlimitedTurns: false, strictPerBoardTurns: false, waitingForGuessers: false };
    const stats = { unityCardsFound: 30, totalUnityCards: 30, totalTurns: 12, assassinCount: 12, score: 2.5, reason: 'all_unity_found' };
    expect(unityEndGameSummary(stats, progress)).toEqual({
      headline: 'Unity solved',
      score: '2.50 Unity cards/turn',
      detail: '30/30 found · 12 turns · 12 assassins · shared pool',
    });
  });
});
