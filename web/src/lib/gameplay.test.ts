import { describe, expect, it } from 'vitest';

import {
  activeMatchLayoutClasses,
  canSubmitClue,
  cardViewState,
  cardContentLabel,
  displayCards,
  defaultGameplayPreferences,
  endGameOutcome,
  buildMemoryCaptureModel,
  cardModeFromImageCount,
  formatClueNumber,
  imageCountForMode,
  cardWordTextClasses,
  cardWordTextSegments,
  cardAspectRatioClasses,
  boardGridClasses,
  boardGridStyle,
  clampBoardColumns,
  clampCardGridMode,
  clampImageCardScale,
  imageCardGridStyle,
  clueLogKey,
  defaultTeamNames,
  displayTeamName,
  isValidHexColor,
  isActiveGuesser,
  parseClueNumber,
  readPanelPreferences,
  readGameplayPreferences,
  shouldAutoJoinRoom,
  shouldCueCardReveal,
  chatCueNotice,
  colorPickerCtaLabel,
  colorSettingsGridClasses,
  teamColorControlClasses,
  modSettingsShellClasses,
  viewerRole,
  bottomShortcutItems,
  ownTeamPlayerNames,
  chatToggleEventName,
  writePanelPreferences,
  writeGameplayPreferences,
  normalizeLobbySettingsForSave,
  type ClueEntry,
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
  { id: 'blueSpy', displayName: 'Blue Spy', team: 'blue', spymaster: true, representative: false, mod: false },
  { id: 'blueGuess', displayName: 'Blue Guess', team: 'blue', spymaster: false, representative: false, mod: false },
  { id: 'redSpy', displayName: 'Red Spy', team: 'red', spymaster: true, representative: false, mod: false },
  { id: 'redGuess', displayName: 'Red Guess', team: 'red', spymaster: false, representative: false, mod: false },
];
const settings: Settings = {
  seed: 1,
  blackCards: 1,
  totalCards: 25,
  autoColorCounts: true,
  blueCards: 9,
  redCards: 8,
  neutralCards: 8,
  wordpackId: 'english',
  enforceClueGuessLimit: false,
  allowInfinityClue: false,
  imageCardCount: 0,
  randomizeTeams: true,
  observerChatEnabled: true,
  mixedImageOrderFirst: false,
};

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
      { id: 'spy1', displayName: 'Spy 1', team: 'blue', spymaster: true, representative: false, mod: false },
      { id: 'spy2', displayName: 'Spy 2', team: 'blue', spymaster: true, representative: false, mod: false },
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
    expect(canSubmitClue(players, viewer('redSpy'), 'blue', 'active', settings).reason).toBe('Only the Libertarians spymaster can clue right now.');
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
  it('defaults local cue/layout preferences and persists partial updates', () => {
    const storage = new MemoryStorage();
    expect(readGameplayPreferences(storage)).toEqual(defaultGameplayPreferences);
    expect(defaultGameplayPreferences.spymasterRevealedStyle).toBe('invisible');

    const saved: GameplayPreferences = { confirmGuesses: false, confirmPasses: true, boardColumnsMobile: 4, boardColumnsDesktop: 5, imageCardScale: 8, strictCardAspectRatios: true, cardGridMode: 'exactAspect', chatSound: false, chatVisualCue: false, cardChoiceSound: false, cardChoiceVisualCue: true, clueSound: true, clueVisualCue: false, endGameSound: true, endGameVisualCue: true, spymasterRevealedStyle: 'greyed' };
    writeGameplayPreferences(storage, saved);
    expect(readGameplayPreferences(storage)).toEqual(saved);

    storage.setItem('codewords.gameplayPreferences', JSON.stringify({ confirmGuesses: false }));
    expect(readGameplayPreferences(storage)).toEqual({ ...defaultGameplayPreferences, confirmGuesses: false });

    storage.setItem('codewords.gameplayPreferences', JSON.stringify({ cardsPerRow: 7 }));
    expect(readGameplayPreferences(storage)).toMatchObject({
      boardColumnsMobile: 7,
      boardColumnsDesktop: 7,
      imageCardScale: 4,
      strictCardAspectRatios: false,
      cardGridMode: 'footprint',
    });
  });

  it('defaults and clamps base board columns, grid mode, plus image-card scale', () => {
    expect(defaultGameplayPreferences).toMatchObject({
      boardColumnsMobile: 4,
      boardColumnsDesktop: 5,
      imageCardScale: 4,
      strictCardAspectRatios: false,
      cardGridMode: 'footprint',
    });
    expect(clampBoardColumns(99)).toBe(13);
    expect(clampBoardColumns(0)).toBe(1);
    expect(clampImageCardScale(1)).toBe(1);
    expect(clampImageCardScale(2)).toBe(2);
    expect(clampImageCardScale(4)).toBe(4);
    expect(clampImageCardScale(8)).toBe(8);
    expect(clampImageCardScale(3)).toBe(4);
    expect(clampCardGridMode('footprint')).toBe('footprint');
    expect(clampCardGridMode('exactAspect')).toBe('exactAspect');
    expect(clampCardGridMode('calibratedRows')).toBe('calibratedRows');
    expect(clampCardGridMode('broken')).toBe('footprint');
  });

  it('persists strict card aspect ratios and maps card type to aspect classes', () => {
    const storage = new MemoryStorage();
    storage.setItem('codewords.gameplayPreferences', JSON.stringify({ strictCardAspectRatios: true }));

    expect(readGameplayPreferences(storage).strictCardAspectRatios).toBe(true);
    expect(cardAspectRatioClasses({ contentType: 'word' }, true)).toBe('aspect-[4/3]');
    expect(cardAspectRatioClasses({ contentType: 'word' }, false)).toBe('min-h-20 sm:min-h-28');
    expect(cardAspectRatioClasses({ contentType: 'image' }, true)).toBe('aspect-[2/3]');
    expect(cardAspectRatioClasses({ contentType: 'image' }, false)).toBe('aspect-[2/3]');
  });

  it('migrates legacy word/image row preferences to base board columns and image scale', () => {
    const storage = new MemoryStorage();
    storage.setItem('codewords.gameplayPreferences', JSON.stringify({
      wordCardsPerRowMobile: 3,
      imageCardsPerRowMobile: 2,
      wordCardsPerRowDesktop: 6,
      imageCardsPerRowDesktop: 2,
    }));

    expect(readGameplayPreferences(storage)).toMatchObject({
      boardColumnsMobile: 3,
      boardColumnsDesktop: 6,
      imageCardScale: 4,
    });
  });

  it('maps image-card scale to grid spans and falls back when columns are narrow', () => {
    expect(imageCardGridStyle({ contentType: 'word' }, 5, 4)).toBe('--card-col-span: 1; --card-row-span: 1;');
    expect(imageCardGridStyle({ contentType: 'image' }, 5, 1)).toBe('--card-col-span: 1; --card-row-span: 1;');
    expect(imageCardGridStyle({ contentType: 'image' }, 5, 2)).toBe('--card-col-span: 1; --card-row-span: 2;');
    expect(imageCardGridStyle({ contentType: 'image' }, 5, 4)).toBe('--card-col-span: 2; --card-row-span: 4;');
    expect(imageCardGridStyle({ contentType: 'image' }, 5, 8)).toBe('--card-col-span: 4; --card-row-span: 8;');
    expect(imageCardGridStyle({ contentType: 'image' }, 1, 4)).toBe('--card-col-span: 1; --card-row-span: 2;');
    expect(imageCardGridStyle({ contentType: 'image' }, 5, 4, 1)).toBe('--card-mobile-col-span: 1; --card-mobile-row-span: 2; --card-col-span: 2; --card-row-span: 4;');
    expect(imageCardGridStyle({ contentType: 'image' }, 5, 1, undefined, 'exactAspect')).toBe('--card-col-span: 1; --card-row-span: 2;');
    expect(imageCardGridStyle({ contentType: 'image' }, 5, 4, undefined, 'exactAspect')).toBe('--card-col-span: 2; --card-row-span: 4;');
    expect(imageCardGridStyle({ contentType: 'image' }, 5, 8, 1, 'exactAspect')).toBe('--card-mobile-col-span: 1; --card-mobile-row-span: 2; --card-col-span: 4; --card-row-span: 8;');
    expect(imageCardGridStyle({ contentType: 'image' }, 5, 4, undefined, 'calibratedRows')).toBe('--card-col-span: 2; --card-row-span: 4;');
  });

  it('builds board grid styles for normal and calibrated row modes', () => {
    expect(boardGridStyle(4, 5, 'footprint')).toBe('--mobile-card-columns: 4; --card-columns: 5; --card-mobile-grid-row: calc((100cqw - 3 * 0.5rem) / 4 * 0.75); --card-grid-row: calc((100cqw - 4 * 0.75rem) / 5 * 0.75);');
    expect(boardGridStyle(4, 5, 'exactAspect')).toBe('--mobile-card-columns: 4; --card-columns: 5; --card-mobile-grid-row: calc((100cqw - 3 * 0.5rem) / 4 * 0.75); --card-grid-row: calc((100cqw - 4 * 0.75rem) / 5 * 0.75);');
    expect(boardGridStyle(4, 5, 'calibratedRows')).toBe('--mobile-card-columns: 4; --card-columns: 5; --card-mobile-grid-row: calc((100cqw - 3 * 0.5rem) / 4 * 0.75); --card-grid-row: calc((100cqw - 4 * 0.75rem) / 5 * 0.75);');
    expect(boardGridClasses('footprint')).toBe('[container-type:inline-size] [grid-auto-rows:var(--card-mobile-grid-row)] md:[grid-auto-rows:var(--card-grid-row)]');
    expect(boardGridClasses('exactAspect')).toBe('[container-type:inline-size] [grid-auto-rows:var(--card-mobile-grid-row)] md:[grid-auto-rows:var(--card-grid-row)]');
    expect(boardGridClasses('calibratedRows')).toBe('[container-type:inline-size] [grid-auto-rows:var(--card-mobile-grid-row)] md:[grid-auto-rows:var(--card-grid-row)]');
  });

  it('defaults collapsible panel preferences open and persists changes', () => {
    const storage = new MemoryStorage();
    expect(readPanelPreferences(storage)).toEqual({ modSettingsOpen: true, localOptionsOpen: true });

    writePanelPreferences(storage, { modSettingsOpen: false, localOptionsOpen: true });
    expect(readPanelPreferences(storage)).toEqual({ modSettingsOpen: false, localOptionsOpen: true });

    storage.setItem('codewords.panelPreferences', '{broken');
    expect(readPanelPreferences(storage)).toEqual({ modSettingsOpen: true, localOptionsOpen: true });
  });
});


describe('bottom control navigation helpers', () => {
  it('exposes working shortcut targets with requested labels', () => {
    expect(bottomShortcutItems.map((item) => `${item.kind}:${item.target}:${item.label}`)).toEqual([
      'board:board:Board',
      'players:players:Players',
      'clues:clues:Clues',
      'settings:settings:Mod Settings',
      'local:local-options:Local Settings',
      'chat:chat:Chat',
    ]);
    expect(chatToggleEventName).toBe('codewords:toggle-chat');
  });

  it('formats the current team row as player names only', () => {
    expect(ownTeamPlayerNames(players, 'blue')).toEqual(['Blue Spy', 'Blue Guess']);
    expect(ownTeamPlayerNames([{ ...players[0], displayName: '' }], 'blue')).toEqual(['Player']);
    expect(ownTeamPlayerNames(players, 'observers')).toEqual([]);
  });
});

describe('board card state', () => {
  const hiddenBlue: GameplayCard = { contentType: 'word', word: 'river', revealed: false, color: 'blue' };
  const revealedRed: GameplayCard = { contentType: 'word', word: 'castle', revealed: true, color: 'red' };
  const hiddenUnknown: GameplayCard = { contentType: 'word', word: 'orbit', revealed: false };

  it('formats image card content labels for confirmations and fallbacks', () => {
    expect(cardContentLabel({ contentType: 'image', imageId: 'abc123', revealed: false })).toBe('Picture card');
    expect(cardContentLabel({ contentType: 'word', word: 'river', revealed: false })).toBe('river');
  });

  it('numbers card badges by display order when images sort first', () => {
    const cards: GameplayCard[] = [
      { contentType: 'word', word: 'alpha', revealed: false },
      { contentType: 'image', imageId: 'img-2', revealed: false },
      { contentType: 'word', word: 'bravo', revealed: false },
      { contentType: 'image', imageId: 'img-4', revealed: false },
    ];

    expect(displayCards(cards, 'mixed', true).map((card) => `${card.contentType}:${card.badgeNumber}:${card.originalIndex}`)).toEqual([
      'image:1:1',
      'image:2:3',
      'word:3:0',
      'word:4:2',
    ]);
    expect(displayCards(cards, 'mixed', false).map((card) => card.badgeNumber)).toEqual([1, 2, 3, 4]);
  });

  it('lets the active board use the full row instead of reserving a right sidebar', () => {
    expect(activeMatchLayoutClasses()).not.toContain('grid-cols-[1fr_24rem]');
    expect(activeMatchLayoutClasses()).toContain('space-y-6');
  });

  it('shrinks word card text and only creates wrap opportunities at spaces or Persian half-spaces', () => {
    expect(cardWordTextClasses('short word')).toContain('whitespace-normal');
    expect(cardWordTextClasses('exceptionally-long-unbroken-card-word')).toContain('overflow-hidden');
    expect(cardWordTextClasses('exceptionally-long-unbroken-card-word')).toContain('text-[clamp');
    expect(cardWordTextSegments('exceptionally-long-unbroken-card-word')).toEqual(['exceptionally-long-unbroken-card-word']);
    expect(cardWordTextSegments('half‌space word')).toEqual(['half‌', 'space ', 'word']);
  });

  it('derives visible color, labels, classes, and last-selected state', () => {
    expect(cardViewState(hiddenUnknown, 0, false, undefined)).toMatchObject({ visibleColor: 'hidden', label: 'Unrevealed', isLastSelected: false });
    expect(cardViewState(hiddenBlue, 1, true, undefined)).toMatchObject({ visibleColor: 'blue', label: 'Blue', isLastSelected: false });
    expect(cardViewState(revealedRed, 2, false, { index: 2, team: 'red' })).toMatchObject({ visibleColor: 'red', label: 'Red', isLastSelected: true });
    expect(cardViewState(revealedRed, 2, false, { index: 2, team: 'red' }).classes).toContain('ring-4');
  });

  it('cues only when a card transitions from unrevealed to revealed', () => {
    const previous: GameplayCard[] = [
      { contentType: 'word', word: 'river', revealed: false, color: 'blue' },
      { contentType: 'word', word: 'castle', revealed: true, color: 'red' },
    ];

    expect(shouldCueCardReveal(previous, previous)).toBe(false);
    expect(shouldCueCardReveal(previous, [
      { ...previous[0], color: 'red' },
      { ...previous[1], color: 'blue' },
    ])).toBe(false);
    expect(shouldCueCardReveal(previous, [
      previous[0],
      { ...previous[1], color: 'blue', revealed: true },
    ])).toBe(false);
    expect(shouldCueCardReveal(previous, [
      { ...previous[0], revealed: true },
      previous[1],
    ])).toBe(true);
  });
});

describe('lobby card count helpers', () => {
  it('derives card content modes from the dynamic total card count', () => {
    expect(cardModeFromImageCount(0, 30)).toBe('words');
    expect(cardModeFromImageCount(30, 30)).toBe('images');
    expect(cardModeFromImageCount(12, 30)).toBe('mixed');

    expect(imageCountForMode('images', 8, 30)).toBe(30);
    expect(imageCountForMode('mixed', 35, 30)).toBe(29);
    expect(imageCountForMode('mixed', 0, 30)).toBe(8);
  });

  it('normalizes lobby card settings before saving', () => {
    const next = normalizeLobbySettingsForSave({
      ...settings,
      totalCards: 30,
      autoColorCounts: false,
      blueCards: 9,
      redCards: 8,
      neutralCards: 1,
      blackCards: 6,
      imageCardCount: 35,
    });

    expect(next.totalCards).toBe(30);
    expect(next.blueCards).toBe(9);
    expect(next.redCards).toBe(8);
    expect(next.neutralCards).toBe(13);
    expect(next.blackCards).toBe(6);
    expect(next.imageCardCount).toBe(30);
  });

  it('keeps automatic team counts independent of a fixed starting team', () => {
    const next = normalizeLobbySettingsForSave({
      ...settings,
      totalCards: 30,
      autoColorCounts: true,
      blueCards: 99,
      redCards: 99,
    });

    expect(next.blueCards).toBe(0);
    expect(next.redCards).toBe(0);
    expect(next.neutralCards).toBe(11);
  });

  it('keeps manual frontend counts within the total before saving', () => {
    const next = normalizeLobbySettingsForSave({
      ...settings,
      totalCards: 12,
      autoColorCounts: false,
      blueCards: 20,
      redCards: 7,
      neutralCards: 8,
      blackCards: 2,
    });

    expect(next.blueCards).toBe(12);
    expect(next.redCards).toBe(0);
    expect(next.neutralCards).toBe(0);
    expect(next.blackCards).toBe(0);
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


describe('regression helpers', () => {
  it('formats configurable team names and validates custom colors', () => {
    expect(defaultTeamNames).toEqual({ blue: 'Libertarians', red: 'Monarchists' });
    expect(displayTeamName('blue', { ...settings, teamNameBlue: 'River Guild' })).toBe('River Guild');
    expect(displayTeamName('red', { ...settings, teamNameRed: '' })).toBe('Monarchists');
    expect(isValidHexColor('#123abc')).toBe(true);
    expect(isValidHexColor('#abc')).toBe(true);
    expect(isValidHexColor('123abc')).toBe(false);
    expect(isValidHexColor('#12zzzz')).toBe(false);
    expect(colorPickerCtaLabel('River Guild', '#14b8a6')).toBe('Choose River Guild color, currently #14b8a6');
  });

  it('uses mobile-safe mod settings and team color layout classes', () => {
    expect(modSettingsShellClasses()).toContain('max-w-full');
    expect(modSettingsShellClasses()).toContain('overflow-hidden');
    expect(colorSettingsGridClasses()).toBe('grid min-w-0 gap-4 md:grid-cols-2');
    expect(teamColorControlClasses()).toContain('flex-col');
    expect(teamColorControlClasses()).toContain('sm:flex-row');
    expect(teamColorControlClasses()).toContain('min-w-0');
  });

  it('treats blank clue numbers as blank instead of NaN numeric values', async () => {
    const { clueNumberFromInput } = await import('./gameplay');
    expect(clueNumberFromInput('')).toEqual({ kind: 'blank' });
    expect(clueNumberFromInput('∞')).toEqual({ kind: 'infinity' });
    expect(clueNumberFromInput('3')).toEqual({ kind: 'numeric', value: 3 });
  });

  it('suppresses chat cues for messages sent by the current viewer', async () => {
    const { shouldCueChatMessage } = await import('./gameplay');
    expect(shouldCueChatMessage({ userId: 'me', playerId: 'me', isHost: false }, { senderUserId: 'me' })).toBe(false);
    expect(shouldCueChatMessage({ userId: 'me', playerId: 'me', isHost: false }, { senderUserId: 'other' })).toBe(true);
    expect(chatCueNotice({ displayName: 'Ada', body: 'abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz' })).toBe('Ada: abcdefghijklmnopqrstuvwxyzabcdefghijkl…');
  });

  it('keeps greyed revealed cards transparent while preserving color hints', () => {
    const view = cardViewState({ contentType: 'word', word: 'castle', revealed: true, color: 'red' }, 0, true, null, 'greyed');
    expect(view.classes).toContain('opacity-45');
    expect(view.classes).not.toContain('grayscale');
    expect(view.classes).toContain('after:bg-current');
  });

  it('builds distinct clue log keys for reset and re-submit rows in the same round', async () => {
    const clues: ClueEntry[] = [
      { round: 1, team: 'blue', text: 'Ocean', number: { kind: 'numeric', value: 2 }, status: 'final', submittedBy: 'blueSpy', updatedBy: 'blueSpy', guesses: 0 },
      { round: 1, team: 'blue', text: 'Forest', number: { kind: 'numeric', value: 2 }, status: 'final', submittedBy: 'blueSpy', updatedBy: 'blueSpy', guesses: 0 },
      { round: 1, team: 'blue', text: 'Forest', number: { kind: 'numeric', value: 2 }, status: 'final', submittedBy: 'blueSpy', updatedBy: 'blueSpy', guesses: 0 },
    ];

    expect(new Set(clues.map((clue, index) => clueLogKey(clue, index))).size).toBe(3);
  });
});


describe('end-game memory and cues', () => {
  const endSettings: Settings = {
    ...settings,
    teamNameBlue: 'River Guild',
    teamNameRed: 'Sun Court',
    customColorBlue: '#2563eb',
    customColorRed: '#dc2626',
    imageCardCount: 1,
    mixedImageOrderFirst: true,
  };
  const endPlayers: LobbyPlayer[] = [
    { id: 'blueSpy', displayName: 'Blue Spy', team: 'blue', spymaster: true, representative: false, mod: false },
    { id: 'blueGuess', displayName: 'Blue Guess', team: 'blue', spymaster: false, representative: false, mod: false },
    { id: 'redSpy', displayName: 'Red Spy', team: 'red', spymaster: true, representative: false, mod: false },
    { id: 'observer', displayName: 'Observer', team: 'observers', spymaster: false, representative: false, mod: false },
  ];
  const endCards: GameplayCard[] = [
    { contentType: 'word', word: 'river', revealed: true, color: 'blue' },
    { contentType: 'image', imageId: 'fox', revealed: true, color: 'red' },
    { contentType: 'word', word: 'shadow', revealed: true, color: 'black' },
  ];

  it('defaults and persists dedicated end-game cue preferences', () => {
    const storage = new MemoryStorage();
    expect(readGameplayPreferences(storage)).toMatchObject({ endGameSound: true, endGameVisualCue: true });

    writeGameplayPreferences(storage, { ...defaultGameplayPreferences, endGameSound: false, endGameVisualCue: false });
    expect(readGameplayPreferences(storage)).toMatchObject({ endGameSound: false, endGameVisualCue: false });

    storage.setItem('codewords.gameplayPreferences', JSON.stringify({ endGameSound: false }));
    expect(readGameplayPreferences(storage)).toMatchObject({ endGameSound: false, endGameVisualCue: true });
  });

  it('classifies the viewer-specific end-game outcome', () => {
    expect(endGameOutcome('blue', viewer('blueGuess'), endPlayers)).toBe('win');
    expect(endGameOutcome('blue', viewer('redSpy'), endPlayers)).toBe('loss');
    expect(endGameOutcome('blue', viewer('observer'), endPlayers)).toBe('neutral');
    expect(endGameOutcome('', viewer('blueGuess'), endPlayers)).toBe('neutral');
  });

  it('builds a capture model with winner, loser, rosters, sorted board, and image fallback labels', () => {
    const model = buildMemoryCaptureModel({
      roomId: 'abc123',
      winner: 'blue',
      players: endPlayers,
      cards: endCards,
      settings: endSettings,
      generatedAt: new Date('2026-05-06T12:00:00.000Z'),
    });

    expect(model.title).toBe('River Guild wins');
    expect(model.subtitle).toBe('Sun Court fell at the final board');
    expect(model.winner.players).toEqual(['Blue Spy', 'Blue Guess']);
    expect(model.loser.players).toEqual(['Red Spy']);
    expect(model.generatedLabel).toContain('May 6, 2026');
    expect(model.cards.map((card) => `${card.badgeNumber}:${card.label}:${card.imageUrl ?? ''}`)).toEqual([
      '1:Picture #1:/api/pictures/fox',
      '2:River:',
      '3:Shadow:',
    ]);
  });
});
