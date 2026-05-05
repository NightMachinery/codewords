import type { RoomSummary, Settings, Viewer } from './api';
import type { LobbyPlayer, Team } from './lobby';

export type GameplayPhase = 'lobby' | 'active' | 'game_over';
export type CardColor = 'blue' | 'red' | 'black' | 'civilian';
export type VisibleCardColor = CardColor | 'hidden';

export interface ClueNumber {
  kind: 'blank' | 'numeric' | 'infinity';
  value?: number;
}

export interface ClueEntry {
  round: number;
  team: 'blue' | 'red';
  text: string;
  number: ClueNumber;
  status: 'active' | 'final' | 'na';
  submittedBy?: string;
  updatedBy?: string;
  guesses: number;
}

export interface LastSelected {
  index: number;
  team: 'blue' | 'red';
}

export interface RemainingCounts {
  blue: number;
  red: number;
  civilian: number;
  black: number;
}

export interface GameplayCard {
  contentType: 'word' | 'image';
  word?: string;
  imageId?: string;
  revealed: boolean;
  color?: CardColor;
}

export type CardMode = 'words' | 'images' | 'mixed';

export interface DisplayCard extends GameplayCard {
  originalIndex: number;
  badgeNumber: number;
}

export interface GameplayPreferences {
  confirmGuesses: boolean;
  confirmPasses: boolean;
  cardsPerRow: number;
  chatSound: boolean;
  chatVisualCue: boolean;
  cardChoiceSound: boolean;
  cardChoiceVisualCue: boolean;
  clueSound: boolean;
  clueVisualCue: boolean;
  spymasterRevealedStyle: 'greyed' | 'invisible' | 'omitted';
}

export interface PanelPreferences {
  modSettingsOpen: boolean;
  localOptionsOpen: boolean;
}

export const gameplayPreferencesStorageKey = 'codewords.gameplayPreferences';
export const panelPreferencesStorageKey = 'codewords.panelPreferences';
export const defaultGameplayPreferences: GameplayPreferences = {
  confirmGuesses: true,
  confirmPasses: false,
  cardsPerRow: 5,
  chatSound: true,
  chatVisualCue: true,
  cardChoiceSound: true,
  cardChoiceVisualCue: true,
  clueSound: true,
  clueVisualCue: true,
  spymasterRevealedStyle: 'invisible',
};

export const defaultPanelPreferences: PanelPreferences = {
  modSettingsOpen: true,
  localOptionsOpen: true,
};

export function viewerId(viewer: Viewer | null | undefined): string {
  return viewer?.playerId || viewer?.userId || '';
}

export function findViewerPlayer(players: LobbyPlayer[], viewer: Viewer | null | undefined): LobbyPlayer | undefined {
  const id = viewerId(viewer);
  return players.find((player) => player.id === id);
}

export function isActiveGuesser(players: LobbyPlayer[], playerId: string | undefined, currentTeam: Team): boolean {
  if (!playerId || (currentTeam !== 'blue' && currentTeam !== 'red')) return false;
  const player = players.find((candidate) => candidate.id === playerId);
  if (!player || player.team !== currentTeam) return false;

  const teammates = players.filter((candidate) => candidate.team === currentTeam);
  if (teammates.length === 0) return false;
  const representatives = teammates.filter((candidate) => candidate.representative);
  if (representatives.length > 0) return player.representative;
  const nonSpymasters = teammates.filter((candidate) => !candidate.spymaster);
  if (nonSpymasters.length === 0) return true;
  return !player.spymaster;
}

export function viewerRole(
  players: LobbyPlayer[],
  viewer: Viewer | null | undefined,
  currentTeam: Team,
  phase: GameplayPhase = 'active',
): {
  kind: 'spectator' | 'player' | 'spymaster';
  team?: Team;
  player?: LobbyPlayer;
  canSeeHiddenColors: boolean;
  activeGuesser: boolean;
} {
  const player = findViewerPlayer(players, viewer);
  const gameOver = phase === 'game_over';
  if (!player) {
    return { kind: 'spectator', canSeeHiddenColors: gameOver, activeGuesser: false };
  }
  const activeGuesser = phase === 'active' && isActiveGuesser(players, player.id, currentTeam);
  return {
    kind: player.spymaster ? 'spymaster' : 'player',
    team: player.team,
    player,
    canSeeHiddenColors: gameOver || player.spymaster,
    activeGuesser,
  };
}

export function canSubmitClue(
  players: LobbyPlayer[],
  viewer: Viewer | null | undefined,
  currentTeam: Team,
  phase: GameplayPhase,
): { allowed: boolean; reason: string } {
  if (phase === 'game_over') return { allowed: false, reason: 'The match is over.' };
  if (phase !== 'active') return { allowed: false, reason: 'Clues are available after the match starts.' };
  const player = findViewerPlayer(players, viewer);
  if (!player) return { allowed: false, reason: 'Spectators are read-only.' };
  if (!player.spymaster) return { allowed: false, reason: 'Only spymasters can clue.' };
  if (player.team !== currentTeam) return { allowed: false, reason: `Only the ${currentTeam} spymaster can clue right now.` };
  return { allowed: true, reason: '' };
}

export function parseClueNumber(value: string): ClueNumber {
  return clueNumberFromInput(value);
}

export function clueNumberFromInput(value: string): ClueNumber {
  const normalized = value.trim();
  if (normalized === '') return { kind: 'blank' };
  if (normalized === '∞' || normalized.toLowerCase() === 'infinity') return { kind: 'infinity' };
  const parsed = Number.parseInt(normalized, 10);
  if (String(parsed) === normalized && parsed >= 1 && parsed <= 9) return { kind: 'numeric', value: parsed };
  return { kind: 'blank' };
}

export function clueNumberValue(number: ClueNumber | undefined): string {
  if (!number || number.kind === 'blank') return '';
  if (number.kind === 'infinity') return '∞';
  return String(number.value ?? '');
}

export function formatClueNumber(number: ClueNumber | undefined): string {
  if (!number || number.kind === 'blank') return 'any';
  if (number.kind === 'infinity') return '∞';
  return String(number.value ?? '');
}

export function clueSubmitProblem(text: string, number: ClueNumber, settings: Settings): string {
  if (!text.trim()) return 'Enter a one-word clue.';
  if (settings.enforceClueGuessLimit && number.kind === 'blank') return 'Pick a clue number when clue limits are enforced.';
  if (!settings.allowInfinityClue && number.kind === 'infinity') return 'Infinity clues are disabled for this room.';
  return '';
}

export function readGameplayPreferences(storage: Pick<Storage, 'getItem'>): GameplayPreferences {
  const raw = storage.getItem(gameplayPreferencesStorageKey);
  if (!raw) return { ...defaultGameplayPreferences };
  try {
    const parsed = JSON.parse(raw) as Partial<GameplayPreferences>;
    const spymasterRevealedStyle = ['greyed', 'invisible', 'omitted'].includes(parsed.spymasterRevealedStyle as string) ? parsed.spymasterRevealedStyle as 'greyed' | 'invisible' | 'omitted' : defaultGameplayPreferences.spymasterRevealedStyle;
    return {
      confirmGuesses: typeof parsed.confirmGuesses === 'boolean' ? parsed.confirmGuesses : defaultGameplayPreferences.confirmGuesses,
      confirmPasses: typeof parsed.confirmPasses === 'boolean' ? parsed.confirmPasses : defaultGameplayPreferences.confirmPasses,
      cardsPerRow: clampCardsPerRow(parsed.cardsPerRow),
      chatSound: typeof parsed.chatSound === 'boolean' ? parsed.chatSound : defaultGameplayPreferences.chatSound,
      chatVisualCue: typeof parsed.chatVisualCue === 'boolean' ? parsed.chatVisualCue : defaultGameplayPreferences.chatVisualCue,
      cardChoiceSound: typeof parsed.cardChoiceSound === 'boolean' ? parsed.cardChoiceSound : defaultGameplayPreferences.cardChoiceSound,
      cardChoiceVisualCue: typeof parsed.cardChoiceVisualCue === 'boolean' ? parsed.cardChoiceVisualCue : defaultGameplayPreferences.cardChoiceVisualCue,
      clueSound: typeof parsed.clueSound === 'boolean' ? parsed.clueSound : defaultGameplayPreferences.clueSound,
      clueVisualCue: typeof parsed.clueVisualCue === 'boolean' ? parsed.clueVisualCue : defaultGameplayPreferences.clueVisualCue,
      spymasterRevealedStyle,
    };
  } catch {
    return { ...defaultGameplayPreferences };
  }
}

export function clampCardsPerRow(value: unknown): number {
  const parsed = typeof value === 'number' ? value : Number.parseInt(String(value ?? ''), 10);
  if (!Number.isFinite(parsed)) return defaultGameplayPreferences.cardsPerRow;
  return Math.min(13, Math.max(1, Math.round(parsed)));
}

export function writeGameplayPreferences(storage: Pick<Storage, 'setItem'>, preferences: GameplayPreferences): void {
  storage.setItem(gameplayPreferencesStorageKey, JSON.stringify(preferences));
}

export function readPanelPreferences(storage: Pick<Storage, 'getItem'>): PanelPreferences {
  const raw = storage.getItem(panelPreferencesStorageKey);
  if (!raw) return { ...defaultPanelPreferences };
  try {
    const parsed = JSON.parse(raw) as Partial<PanelPreferences>;
    return {
      modSettingsOpen: typeof parsed.modSettingsOpen === 'boolean' ? parsed.modSettingsOpen : defaultPanelPreferences.modSettingsOpen,
      localOptionsOpen: typeof parsed.localOptionsOpen === 'boolean' ? parsed.localOptionsOpen : defaultPanelPreferences.localOptionsOpen,
    };
  } catch {
    return { ...defaultPanelPreferences };
  }
}

export function writePanelPreferences(storage: Pick<Storage, 'setItem'>, preferences: PanelPreferences): void {
  storage.setItem(panelPreferencesStorageKey, JSON.stringify(preferences));
}

export function shouldAutoJoinRoom(room: RoomSummary, credentialMode: 'auth' | 'migrate' | 'none', displayName: string): boolean {
  return credentialMode === 'auth' && room.status === 'lobby' && displayName.trim().length > 0;
}

export function cardContentLabel(card: GameplayCard): string {
  if (card.contentType === 'image') return 'Picture card';
  return card.word || 'Card';
}

export function cardImageUrl(card: GameplayCard): string {
  return card.contentType === 'image' && card.imageId ? `/api/pictures/${encodeURIComponent(card.imageId)}` : '';
}

export function toTitleCase(str: string | undefined): string {
  if (!str) return '';
  return str.toLowerCase().split(' ').map(word => word.charAt(0).toUpperCase() + word.slice(1)).join(' ');
}

export function hexWithAlpha(hex: string | undefined, alpha: string): string {
  if (!hex || !hex.startsWith('#')) return hex ?? '';
  if (hex.length === 4) {
    const [, r, g, b] = hex;
    return `#${r}${r}${g}${g}${b}${b}${alpha}`;
  }
  if (hex.length === 7) return `${hex}${alpha}`;
  return '';
}

export function cardWordTextClasses(word: string | undefined): string {
  const length = [...(word ?? '')].length;
  const size = length > 28 ? 'text-sm sm:text-base' : length > 18 ? 'text-base sm:text-lg' : 'text-xl sm:text-2xl';
  return ['mt-4 block overflow-hidden break-normal hyphens-auto text-balance text-center font-black tracking-[0.04em]', size].join(' ');
}

export function cardModeFromImageCount(imageCardCount: number): CardMode {
  if (imageCardCount <= 0) return 'words';
  if (imageCardCount >= 25) return 'images';
  return 'mixed';
}

export function imageCountForMode(mode: CardMode, currentMixedCount: number): number {
  if (mode === 'words') return 0;
  if (mode === 'images') return 25;
  return Math.min(24, Math.max(1, currentMixedCount || 8));
}

export function displayCards(cards: GameplayCard[], cardMode: CardMode, imageOrderFirst: boolean): DisplayCard[] {
  const list = cards.map((card, index) => ({ ...card, originalIndex: index, badgeNumber: index + 1 }));
  const ordered = cardMode !== 'mixed' || !imageOrderFirst ? list : list.sort((a, b) => {
    if (a.contentType === 'image' && b.contentType !== 'image') return -1;
    if (a.contentType !== 'image' && b.contentType === 'image') return 1;
    return a.originalIndex - b.originalIndex;
  });
  return ordered.map((card, index) => ({ ...card, badgeNumber: index + 1 }));
}

export function shouldCueCardReveal(previousCards: GameplayCard[], nextCards: GameplayCard[]): boolean {
  return nextCards.some((card, index) => card.revealed && !previousCards[index]?.revealed);
}

export function clueLogKey(clue: ClueEntry, displayIndex: number): string {
  return [
    clue.round,
    clue.team,
    clue.status,
    clue.text,
    formatClueNumber(clue.number),
    clue.guesses,
    clue.submittedBy ?? '',
    clue.updatedBy ?? '',
    displayIndex,
  ].join(':');
}

export function cardViewState(
  card: GameplayCard,
  index: number,
  showHiddenColor: boolean,
  lastSelected: LastSelected | null | undefined,
  revealedStyle: 'normal' | 'greyed' | 'invisible' | 'omitted' = 'normal'
): {
  visibleColor: VisibleCardColor;
  label: string;
  isLastSelected: boolean;
  classes: string;
} {
  const visibleColor: VisibleCardColor = card.revealed || showHiddenColor ? (card.color ?? 'hidden') : 'hidden';
  const isLastSelected = lastSelected?.index === index;
  const colorClass = {
    hidden: 'border-slate-700 bg-slate-900 text-slate-100 hover:border-emerald-200',
    blue: 'border-blue-300/70 bg-blue-500/25 text-blue-50',
    red: 'border-red-300/70 bg-red-500/25 text-red-50',
    black: 'border-zinc-300/60 bg-zinc-950 text-zinc-50',
    civilian: 'border-amber-100/60 bg-amber-100/20 text-amber-50',
  }[visibleColor];

  let styleClasses = '';
  if (card.revealed) {
    if (revealedStyle === 'normal') styleClasses = 'opacity-95';
    else if (revealedStyle === 'greyed') styleClasses = 'opacity-45 saturate-75 contrast-90 after:pointer-events-none after:absolute after:right-2 after:top-2 after:h-2.5 after:w-2.5 after:rounded-full after:bg-current after:opacity-70';
    else if (revealedStyle === 'invisible') styleClasses = 'invisible';
    else if (revealedStyle === 'omitted') styleClasses = 'hidden';
  }

  return {
    visibleColor,
    label: visibleColor === 'hidden' ? 'Unrevealed' : visibleColor[0].toUpperCase() + visibleColor.slice(1),
    isLastSelected,
    classes: [colorClass, styleClasses, showHiddenColor && !card.revealed ? 'shadow-inner shadow-white/10' : '', isLastSelected ? 'ring-4 ring-emerald-200' : ''].filter(Boolean).join(' '),
  };
}


export function shouldCueChatMessage(viewer: Viewer | null | undefined, message: Pick<{ senderUserId: string }, 'senderUserId'>): boolean {
  const id = viewerId(viewer);
  return !id || message.senderUserId !== id;
}
