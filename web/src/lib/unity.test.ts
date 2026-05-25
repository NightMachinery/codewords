import { describe, expect, it } from 'vitest';

import {
  defaultSettings,
  type Settings,
} from './api';
import {
  defaultGameplayPreferences,
  defaultTeamNames,
  displayTeamName,
  isActiveGuesser,
  normalizeLobbySettingsForSave,
  playerRoleBadgeKinds,
  teamColor,
  shouldCueUnitySpymaster,
  unityBoardViewCards,
  unityEndGameSummary,
  unityPlayerBoardRows,
  unityGuessDisabledReason,
  isUnityTemporaryRepresentative,
  unityTurnsPendingForDisplayedBoard,
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
    expect(settings.unityTurnLimit).toBe(6);
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
    expect(isActiveGuesser(ownerRep, 'p2', 'unity', 'host', 'p2')).toBe(true);
    expect(isActiveGuesser(ownerRep, 'p3', 'unity', 'host', 'p2')).toBe(false);

    const twoReps = ownerRep.map((player) => player.id === 'p2' ? { ...player, representative: true } : player);
    expect(isActiveGuesser(twoReps, 'p2', 'unity', 'host')).toBe(true);
    expect(isActiveGuesser(twoReps, 'p3', 'unity', 'host')).toBe(false);
  });

  it('marks only implicit unity guessers as temporary representatives', () => {
    expect(isUnityTemporaryRepresentative(players, 'p2', 'host')).toBe(true);
    expect(isUnityTemporaryRepresentative(players, 'host', 'host')).toBe(false);

    const oneRep = players.map((player) => player.id === 'p2' ? { ...player, representative: true } : player);
    expect(isUnityTemporaryRepresentative(oneRep, 'p2', 'host')).toBe(false);
    expect(isUnityTemporaryRepresentative(oneRep, 'p3', 'host')).toBe(false);

    const ownerRep = players.map((player) => player.id === 'host' ? { ...player, representative: true } : player);
    expect(isUnityTemporaryRepresentative(ownerRep, 'p2', 'host', 'p2')).toBe(true);
    expect(isUnityTemporaryRepresentative(ownerRep, 'p3', 'host', 'p2')).toBe(false);

    const twoReps = ownerRep.map((player) => player.id === 'p2' ? { ...player, representative: true } : player);
    expect(isUnityTemporaryRepresentative(twoReps, 'p2', 'host')).toBe(false);
    expect(isUnityTemporaryRepresentative(twoReps, 'p3', 'host')).toBe(false);
  });

  it('selects active or own unity board cards for display', () => {
    const active: UnityBoardSnapshot = { ownerId: 'host', cards: [{ contentType: 'word', word: 'Active', revealed: false }], clueLog: [], turnsUsed: 1, remainingCounts: { unity: 9, civilian: 10, black: 4 } };
    const previous: UnityBoardSnapshot = { ownerId: 'p3', cards: [{ contentType: 'word', word: 'Previous', revealed: true, color: 'civilian' }], clueLog: [], turnsUsed: 1, remainingCounts: { unity: 8, civilian: 9, black: 4 } };
    const own: UnityBoardSnapshot = { ownerId: 'p2', cards: [{ contentType: 'word', word: 'Own', revealed: false, color: 'unity' }], clueLog: [], turnsUsed: 0, remainingCounts: { unity: 10, civilian: 11, black: 4 } };

    expect(unityBoardViewCards('active', active, own, previous)[0].word).toBe('Active');
    expect(unityBoardViewCards('previous', active, own, previous)[0].word).toBe('Previous');
    expect(unityBoardViewCards('own', active, own, previous)[0].word).toBe('Own');
    expect(unityBoardViewCards('own', active, null, previous)[0].word).toBe('Active');
  });

  it('defaults spy revealed cards to greyed', () => {
    expect(defaultGameplayPreferences.spymasterRevealedStyle).toBe('greyed');
  });

  it('cues unity spymasters only for active live board handoffs', () => {
    const board: UnityBoardSnapshot = { ownerId: 'host', cards: [{ contentType: 'word', word: 'Active', revealed: false }], clueLog: [], turnsUsed: 0, remainingCounts: { unity: 10, civilian: 11, black: 4 } };

    expect(shouldCueUnitySpymaster({
      mode: 'unity',
      phase: 'lobby',
      viewerId: 'host',
      previousActiveBoardOwner: '',
      nextActiveBoard: board,
    })).toBe(false);

    expect(shouldCueUnitySpymaster({
      mode: 'unity',
      phase: 'active',
      viewerId: 'host',
      previousActiveBoardOwner: '',
      nextActiveBoard: { ...board, cards: [] },
    })).toBe(false);

    expect(shouldCueUnitySpymaster({
      mode: 'unity',
      phase: 'active',
      viewerId: 'host',
      previousActiveBoardOwner: 'p2',
      nextActiveBoard: board,
    })).toBe(true);

    expect(shouldCueUnitySpymaster({
      mode: 'unity',
      phase: 'active',
      viewerId: 'host',
      previousActiveBoardOwner: 'host',
      nextActiveBoard: board,
    })).toBe(false);
  });

  it('blocks active guessing while viewing a different own board', () => {
    expect(unityGuessDisabledReason({
      phase: 'active',
      hasPlayer: true,
      waitingForGuessers: false,
      activeGuesser: true,
      playerId: 'p2',
      activeBoardOwner: 'host',
      boardView: 'own',
      activeBoardId: 'host',
      displayedBoardId: 'p2',
      transitionLocked: false,
      enforceClueGuessLimit: false,
      currentClue: null,
      cardRevealed: false,
    })).toBe('Switch to the active board to reveal cards.');

    expect(unityGuessDisabledReason({
      phase: 'active',
      hasPlayer: true,
      waitingForGuessers: false,
      activeGuesser: true,
      playerId: 'p2',
      activeBoardOwner: 'host',
      boardView: 'previous',
      activeBoardId: 'host',
      displayedBoardId: 'p3',
      transitionLocked: true,
      enforceClueGuessLimit: false,
      currentClue: null,
      cardRevealed: false,
    })).toBe('Waiting for the next Unity board.');
  });

  it('formats unity end game stats and shared pool progress', () => {
    const progress: UnityProgress = { unityCardsFound: 17, totalUnityCards: 30, sharedTurnsRemaining: 3, unlimitedTurns: false, strictPerBoardTurns: false, waitingForGuessers: false };
    const stats = { unityCardsFound: 30, totalUnityCards: 30, totalTurns: 12, assassinCount: 12, score: 2.5, reason: 'all_unity_found', boardStats: [] };
    expect(unityEndGameSummary(stats, progress)).toEqual({
      headline: 'Unification successful.',
      score: '2.50 Unity cards/turn',
      detail: '30/30 found · 12 turns · 12 assassins · shared pool',
    });

    expect(unityEndGameSummary({ ...stats, unityCardsFound: 12, reason: 'assassin' }, progress).headline).toBe('Players were divided.');
  });

  it('formats unity player board rows with N/A averages for unplayed boards', () => {
    const rows = unityPlayerBoardRows(players, [
      { ownerId: 'host', unityCardsFound: 4, totalUnityCards: 10, turnsUsed: 2, unityCardsPerTurn: 2 },
      { ownerId: 'p2', unityCardsFound: 0, totalUnityCards: 10, turnsUsed: 0, unityCardsPerTurn: null },
    ]);

    expect(rows).toEqual([
      { id: 'host', name: 'Host', status: 'board', detail: '4/10 Unity · 2.00/turn · 2 turns' },
      { id: 'p2', name: 'P2', status: 'board', detail: '0/10 Unity · N/A · 0 turns' },
      { id: 'p3', name: 'P3', status: 'no-board', detail: 'No Unity board' },
    ]);
  });

  it('sorts unity player board rows by higher average first', () => {
    const rows = unityPlayerBoardRows(players, [
      { ownerId: 'host', unityCardsFound: 1, totalUnityCards: 10, turnsUsed: 2, unityCardsPerTurn: 0.5 },
      { ownerId: 'p2', unityCardsFound: 3, totalUnityCards: 10, turnsUsed: 2, unityCardsPerTurn: 1.5 },
      { ownerId: 'p3', unityCardsFound: 0, totalUnityCards: 10, turnsUsed: 0, unityCardsPerTurn: null },
    ]);

    expect(rows.map((row) => row.id)).toEqual(['p2', 'host', 'p3']);
  });

  it('shows spy instead of rep when both role badges apply', () => {
    expect(playerRoleBadgeKinds({ ...players[0], spymaster: true, representative: true, mod: true })).toEqual(['spy', 'mod']);
    expect(playerRoleBadgeKinds({ ...players[1], temporaryRepresentative: true })).toEqual(['tempRep']);
  });

  it('marks active finite unity board turns as pending before consumption', () => {
    const active: UnityBoardSnapshot = { ownerId: 'host', cards: [{ contentType: 'word', word: 'Active', revealed: false }], clueLog: [], turnsUsed: 0, remainingCounts: { unity: 10, civilian: 11, black: 4 } };
    const previous: UnityBoardSnapshot = { ownerId: 'p2', cards: [{ contentType: 'word', word: 'Previous', revealed: false }], clueLog: [], turnsUsed: 1, remainingCounts: { unity: 9, civilian: 10, black: 4 } };

    expect(unityTurnsPendingForDisplayedBoard({
      phase: 'active',
      unlimitedTurns: false,
      displayedBoardId: 'host',
      activeBoardId: 'host',
      transitionLocked: false,
    })).toBe(true);
    expect(unityTurnsPendingForDisplayedBoard({
      phase: 'active',
      unlimitedTurns: false,
      displayedBoardId: previous.ownerId,
      activeBoardId: active.ownerId,
      transitionLocked: false,
    })).toBe(false);
    expect(unityTurnsPendingForDisplayedBoard({
      phase: 'active',
      unlimitedTurns: true,
      displayedBoardId: active.ownerId,
      activeBoardId: active.ownerId,
      transitionLocked: false,
    })).toBe(false);
    expect(unityTurnsPendingForDisplayedBoard({
      phase: 'game_over',
      unlimitedTurns: false,
      displayedBoardId: active.ownerId,
      activeBoardId: active.ownerId,
      transitionLocked: false,
    })).toBe(false);
  });
});
