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

export const minTotalCards = 9;
export const maxTotalCards = 100;
export const defaultTotalCards = 25;

export interface DisplayCard extends GameplayCard {
  originalIndex: number;
  badgeNumber: number;
}

export type ImageCardScale = 1 | 2 | 4 | 8;

export interface GameplayPreferences {
  confirmGuesses: boolean;
  confirmPasses: boolean;
  boardColumnsMobile: number;
  boardColumnsDesktop: number;
  imageCardScale: ImageCardScale;
  strictCardAspectRatios: boolean;
  boardMustFitHeight: boolean;
  showNumberBadges: boolean;
  chatSound: boolean;
  chatVisualCue: boolean;
  cardChoiceSound: boolean;
  cardChoiceVisualCue: boolean;
  clueSound: boolean;
  clueVisualCue: boolean;
  endGameSound: boolean;
  endGameVisualCue: boolean;
  spymasterRevealedStyle: 'greyed' | 'invisible' | 'omitted';
}

export interface PanelPreferences {
  modSettingsOpen: boolean;
  localOptionsOpen: boolean;
}

export type BottomShortcutKind = 'board' | 'players' | 'clues' | 'settings' | 'local' | 'chat';

export interface BottomShortcutItem {
  kind: BottomShortcutKind;
  target: string;
  label: string;
}

export const chatToggleEventName = 'codewords:toggle-chat';

export const bottomShortcutItems: BottomShortcutItem[] = [
  { kind: 'board', target: 'board', label: 'Board' },
  { kind: 'players', target: 'players', label: 'Players' },
  { kind: 'clues', target: 'clues', label: 'Clues' },
  { kind: 'settings', target: 'settings', label: 'Mod Settings' },
  { kind: 'local', target: 'local-options', label: 'Local Settings' },
  { kind: 'chat', target: 'chat', label: 'Chat' },
];

export function filteredBottomShortcutItems(canUseModSettings: boolean): BottomShortcutItem[] {
  return bottomShortcutItems.filter((item) => canUseModSettings || item.target !== 'settings');
}

export function ownTeamPlayerNames(players: LobbyPlayer[], team: Team | undefined): string[] {
  if (team !== 'blue' && team !== 'red') return [];
  return players
    .filter((player) => player.team === team)
    .map((player) => player.displayName.trim() || 'Player');
}

export function sortedTurnPlayers(players: LobbyPlayer[], team: Team | undefined): LobbyPlayer[] {
  if (team !== 'blue' && team !== 'red') return [];
  const roleRank = (player: LobbyPlayer): number => {
    if (player.spymaster) return 0;
    if (player.representative) return 1;
    return 2;
  };
  return players
    .map((player, index) => ({ player, index }))
    .filter(({ player }) => player.team === team)
    .sort((left, right) => roleRank(left.player) - roleRank(right.player) || left.index - right.index)
    .map(({ player }) => player);
}

export const gameplayPreferencesStorageKey = 'codewords.gameplayPreferences';
export const panelPreferencesStorageKey = 'codewords.panelPreferences';
export const defaultGameplayPreferences: GameplayPreferences = {
  confirmGuesses: true,
  confirmPasses: false,
  boardColumnsMobile: 4,
  boardColumnsDesktop: 8,
  imageCardScale: 4,
  strictCardAspectRatios: true,
  boardMustFitHeight: true,
  showNumberBadges: true,
  chatSound: true,
  chatVisualCue: true,
  cardChoiceSound: true,
  cardChoiceVisualCue: true,
  clueSound: true,
  clueVisualCue: true,
  endGameSound: true,
  endGameVisualCue: true,
  spymasterRevealedStyle: 'invisible',
};

export const defaultTeamNames = {
  blue: 'Libertarians',
  red: 'Monarchists',
} as const;

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
  settings?: Settings,
): { allowed: boolean; reason: string } {
  if (phase === 'game_over') return { allowed: false, reason: 'The match is over.' };
  if (phase !== 'active') return { allowed: false, reason: 'Clues are available after the match starts.' };
  const player = findViewerPlayer(players, viewer);
  if (!player) return { allowed: false, reason: 'Spectators are read-only.' };
  if (!player.spymaster) return { allowed: false, reason: 'Only spymasters can clue.' };
  if (player.team !== currentTeam) return { allowed: false, reason: `Only the ${displayTeamName(currentTeam, settings)} spymaster can clue right now.` };
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
    const parsed = JSON.parse(raw) as Partial<GameplayPreferences> & { cardsPerRow?: unknown; wordCardsPerRowMobile?: unknown; wordCardsPerRowDesktop?: unknown; imageCardsPerRowMobile?: unknown; imageCardsPerRowDesktop?: unknown };
    const spymasterRevealedStyle = ['greyed', 'invisible', 'omitted'].includes(parsed.spymasterRevealedStyle as string) ? parsed.spymasterRevealedStyle as 'greyed' | 'invisible' | 'omitted' : defaultGameplayPreferences.spymasterRevealedStyle;
    return {
      confirmGuesses: typeof parsed.confirmGuesses === 'boolean' ? parsed.confirmGuesses : defaultGameplayPreferences.confirmGuesses,
      confirmPasses: typeof parsed.confirmPasses === 'boolean' ? parsed.confirmPasses : defaultGameplayPreferences.confirmPasses,
      boardColumnsMobile: clampBoardColumns(parsed.boardColumnsMobile ?? parsed.wordCardsPerRowMobile ?? parsed.cardsPerRow, defaultGameplayPreferences.boardColumnsMobile),
      boardColumnsDesktop: clampBoardColumns(parsed.boardColumnsDesktop ?? parsed.wordCardsPerRowDesktop ?? parsed.cardsPerRow, defaultGameplayPreferences.boardColumnsDesktop),
      imageCardScale: clampImageCardScale(parsed.imageCardScale),
      strictCardAspectRatios: typeof parsed.strictCardAspectRatios === 'boolean' ? parsed.strictCardAspectRatios : defaultGameplayPreferences.strictCardAspectRatios,
      boardMustFitHeight: typeof parsed.boardMustFitHeight === 'boolean' ? parsed.boardMustFitHeight : defaultGameplayPreferences.boardMustFitHeight,
      showNumberBadges: typeof parsed.showNumberBadges === 'boolean' ? parsed.showNumberBadges : defaultGameplayPreferences.showNumberBadges,
      chatSound: typeof parsed.chatSound === 'boolean' ? parsed.chatSound : defaultGameplayPreferences.chatSound,
      chatVisualCue: typeof parsed.chatVisualCue === 'boolean' ? parsed.chatVisualCue : defaultGameplayPreferences.chatVisualCue,
      cardChoiceSound: typeof parsed.cardChoiceSound === 'boolean' ? parsed.cardChoiceSound : defaultGameplayPreferences.cardChoiceSound,
      cardChoiceVisualCue: typeof parsed.cardChoiceVisualCue === 'boolean' ? parsed.cardChoiceVisualCue : defaultGameplayPreferences.cardChoiceVisualCue,
      clueSound: typeof parsed.clueSound === 'boolean' ? parsed.clueSound : defaultGameplayPreferences.clueSound,
      clueVisualCue: typeof parsed.clueVisualCue === 'boolean' ? parsed.clueVisualCue : defaultGameplayPreferences.clueVisualCue,
      endGameSound: typeof parsed.endGameSound === 'boolean' ? parsed.endGameSound : defaultGameplayPreferences.endGameSound,
      endGameVisualCue: typeof parsed.endGameVisualCue === 'boolean' ? parsed.endGameVisualCue : defaultGameplayPreferences.endGameVisualCue,
      spymasterRevealedStyle,
    };
  } catch {
    return { ...defaultGameplayPreferences };
  }
}

export function clampCardsPerRow(value: unknown): number {
  return clampBoardColumns(value, defaultGameplayPreferences.boardColumnsDesktop);
}

export function clampBoardColumns(value: unknown, fallback = 5): number {
  const parsed = typeof value === 'number' ? value : Number.parseInt(String(value ?? ''), 10);
  if (!Number.isFinite(parsed)) return fallback;
  return Math.min(13, Math.max(1, Math.round(parsed)));
}

export function clampImageCardScale(value: unknown): ImageCardScale {
  return value === 1 || value === 2 || value === 4 || value === 8 ? value : defaultGameplayPreferences.imageCardScale;
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
  if (!isValidHexColor(hex)) return '';
  const color = hex as string;
  if (color.length === 4) {
    const [, r, g, b] = color;
    return `#${r}${r}${g}${g}${b}${b}${alpha}`;
  }
  if (color.length === 7) return `${color}${alpha}`;
  return '';
}

export function isValidHexColor(hex: string | undefined): boolean {
  return Boolean(hex && (/^#[0-9a-fA-F]{3}$/.test(hex) || /^#[0-9a-fA-F]{6}$/.test(hex)));
}

export function normalizedHexColor(hex: string | undefined, fallback = ''): string {
  return isValidHexColor(hex) ? hex as string : fallback;
}

export function colorPickerCtaLabel(teamName: string, color: string): string {
  return `Choose ${teamName} color, currently ${color}`;
}

export function modSettingsShellClasses(): string {
  return 'max-w-full overflow-hidden rounded-[2rem] border border-slate-700/70 bg-slate-900/80 p-4 shadow-2xl shadow-slate-950/30 sm:p-6';
}

export function colorSettingsGridClasses(): string {
  return 'grid min-w-0 gap-4 md:grid-cols-2';
}

export function teamColorControlClasses(): string {
  return 'mt-2 grid min-w-0 grid-cols-1 items-stretch gap-2 overflow-hidden rounded-2xl border bg-slate-950/60 p-2 shadow-inner shadow-slate-950/30';
}

export function roomMainClasses(): string {
  return 'min-h-screen w-full overflow-x-hidden bg-[oklch(14%_0.018_255)] pb-32 font-sans text-slate-100';
}

export function lobbyStartPanelClasses(): string {
  return 'fixed bottom-0 left-0 right-0 z-30 border-t border-slate-700/60 bg-slate-900/92 p-3 shadow-[0_-10px_30px_rgba(0,0,0,0.45)] backdrop-blur-md';
}

export function pressableButtonClasses(classes = ''): string {
  return [
    classes,
    'transition active:translate-y-px focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-200/80 focus-visible:ring-offset-2 focus-visible:ring-offset-slate-950 disabled:active:translate-y-0',
  ].filter(Boolean).join(' ');
}

export function displayTeamName(team: Team | 'blue' | 'red' | '', settings: Settings | undefined): string {
  if (team === 'blue') return (settings?.teamNameBlue?.trim() || defaultTeamNames.blue).slice(0, 30);
  if (team === 'red') return (settings?.teamNameRed?.trim() || defaultTeamNames.red).slice(0, 30);
  if (team === 'observers') return 'Observers';
  return 'Waiting';
}

export function teamColor(team: Team | 'blue' | 'red' | '', settings: Settings | undefined): string {
  if (team === 'blue') return normalizedHexColor(settings?.customColorBlue, '#3b82f6');
  if (team === 'red') return normalizedHexColor(settings?.customColorRed, '#ef4444');
  return '';
}

export function boardGridStyle(mobileColumns: number, columns: number): string {
  const safeMobileColumns = clampBoardColumns(mobileColumns, defaultGameplayPreferences.boardColumnsMobile);
  const safeColumns = clampBoardColumns(columns, defaultGameplayPreferences.boardColumnsDesktop);
  const baseVars = `--mobile-card-columns: ${safeMobileColumns}; --card-columns: ${safeColumns};`;
  return `${baseVars} --card-mobile-grid-row: calc(((100cqw - ${safeMobileColumns - 1} * 0.5rem) / ${safeMobileColumns} * 0.75) - 0.25rem); --card-grid-row: calc(((100cqw - ${safeColumns - 1} * 0.75rem) / ${safeColumns} * 0.75) - 0.375rem);`;
}

export function boardGridContainerClasses(): string {
  return '[container-type:inline-size]';
}

export function boardFitAvailableHeight(input: { viewportHeight: number; topReservedHeight: number; bottomPanelHeight: number; verticalMargin?: number }): number {
  const margin = input.verticalMargin ?? 24;
  return Math.max(0, Math.floor(input.viewportHeight - input.topReservedHeight - input.bottomPanelHeight - margin));
}

export function boardFitHeightStyle(input: {
  enabled: boolean;
  isMobile: boolean;
  viewportHeight: number;
  topReservedHeight: number;
  bottomPanelHeight: number;
  boardNaturalWidth: number;
  boardNaturalHeight: number;
  currentBoardWidth?: number;
  currentBoardHeight?: number;
  scrollY?: number;
  verticalMargin?: number;
}): string {
  if (!input.enabled || input.isMobile || input.boardNaturalWidth <= 0 || input.boardNaturalHeight <= 0) return '';
  const availableHeight = boardFitAvailableHeight(input);
  if (availableHeight <= 0 || input.boardNaturalHeight <= availableHeight) return '';
  const ratio = availableHeight / input.boardNaturalHeight;
  const width = Math.max(320, Math.floor(input.boardNaturalWidth * ratio));
  return `max-width: ${width}px; margin-left: auto; margin-right: auto;`;
}

export function cardChromeClasses(card: Pick<DisplayCard, 'contentType'>, _isLastSelected: boolean): string {
  if (card.contentType === 'image') return 'overflow-hidden border-0 p-2';
  return 'overflow-hidden p-1';
}

export function cardChromeStyle(card: Pick<DisplayCard, 'contentType'>, visibleColor: VisibleCardColor, customColor: string, isLastSelected: boolean): string {
  if (visibleColor === 'hidden' || !customColor) return '';
  if (card.contentType === 'image') return `background-color: ${hexWithAlpha(customColor, 'B3')}; color: white`;
  const borderColor = isLastSelected ? 'transparent' : hexWithAlpha(customColor, 'B3');
  return `border-color: ${borderColor}; background-color: ${hexWithAlpha(customColor, '40')}; color: white`;
}

export function cardDisabledStateClasses(input: { disabled: boolean; revealed?: boolean; revealedStyle: 'normal' | 'greyed' | 'invisible' | 'omitted' }): string {
  if (!input.disabled) return '';
  return input.revealed && input.revealedStyle === 'greyed' ? 'disabled:opacity-30' : 'disabled:opacity-80';
}

export function imageColorFrameClasses(isLastSelected: boolean): string {
  return isLastSelected ? 'pointer-events-none absolute inset-[4px] z-20 rounded-lg border-[12px]' : '';
}

export function selectedImageOverlayStyle(_visibleColor: VisibleCardColor, _customColor: string): string {
  return 'border-color: rgba(167, 243, 208, 0.95);';
}

export function boardGridClasses(): string {
  return '[grid-auto-rows:var(--card-mobile-grid-row)] md:[grid-auto-rows:var(--card-grid-row)]';
}

export function boardGridLayoutClasses(captureMode = false): string {
  if (captureMode) {
    return 'grid grid-flow-dense gap-3 [grid-template-columns:repeat(var(--card-columns),minmax(0,1fr))] [grid-auto-rows:var(--card-grid-row)]';
  }
  return ['grid grid-flow-dense gap-2 md:gap-3 [grid-template-columns:repeat(var(--mobile-card-columns),minmax(0,1fr))] md:[grid-template-columns:repeat(var(--card-columns),minmax(0,1fr))]', boardGridClasses()].join(' ');
}

export function boardCardSpanClasses(captureMode = false): string {
  if (captureMode) return 'col-span-[var(--card-col-span)] row-span-[var(--card-row-span)]';
  return 'col-span-[var(--card-mobile-col-span)] row-span-[var(--card-mobile-row-span)] md:col-span-[var(--card-col-span)] md:row-span-[var(--card-row-span)]';
}

export function imageCardGridStyle(card: Pick<DisplayCard, 'contentType'>, columns: number, scale: ImageCardScale, mobileColumns?: number): string {
  const desktopSpan = cardGridSpan(card, columns, scale);
  const desktopVars = `--card-col-span: ${desktopSpan.columns}; --card-row-span: ${desktopSpan.rows};`;
  if (mobileColumns === undefined) return desktopVars;
  const mobileSpan = cardGridSpan(card, mobileColumns, scale);
  return `--card-mobile-col-span: ${mobileSpan.columns}; --card-mobile-row-span: ${mobileSpan.rows}; ${desktopVars}`;
}

export function cardAspectRatioClasses(card: Pick<DisplayCard, 'contentType'>, strictCardAspectRatios: boolean): string {
  if (card.contentType === 'image') return 'aspect-[2/3]';
  return strictCardAspectRatios ? 'h-full' : 'min-h-20 sm:min-h-28';
}

function cardGridSpan(card: Pick<DisplayCard, 'contentType'>, columns: number, scale: ImageCardScale): { columns: number; rows: number } {
  if (card.contentType !== 'image') return { columns: 1, rows: 1 };
  const safeColumns = clampBoardColumns(columns);
  const requestedScale = clampImageCardScale(scale);
  const requested = imageSpanForScale(requestedScale);
  const span = requested.columns <= safeColumns ? requested : imageSpanForScale(2);
  return { columns: span.columns, rows: Math.max(span.rows, span.columns * 2) };
}

function imageSpanForScale(scale: ImageCardScale): { columns: number; rows: number } {
  if (scale === 1) return { columns: 1, rows: 1 };
  if (scale === 2) return { columns: 1, rows: 2 };
  if (scale === 4) return { columns: 2, rows: 4 };
  return { columns: 4, rows: 8 };
}

export function activeMatchLayoutClasses(): string {
  return 'space-y-6';
}

export function cardWordTextSegments(word: string | undefined): string[] {
  const value = word ?? '';
  if (!value) return [''];

  const segments: string[] = [];
  let segment = '';
  for (const char of value) {
    segment += char;
    if (/\s/u.test(char) || char === '\u200c') {
      segments.push(segment);
      segment = '';
    }
  }
  if (segment) segments.push(segment);
  return segments;
}

export function cardWordTextClasses(word: string | undefined): string {
  const length = [...(word ?? '')].length;
  const size = length > 44
    ? 'text-[clamp(0.42rem,1.7cqw,0.72rem)]'
    : length > 32
      ? 'text-[clamp(0.5rem,2.6cqw,0.95rem)]'
      : length > 22
        ? 'text-[clamp(0.68rem,4.8cqw,1.18rem)]'
        : length > 14
          ? 'text-[clamp(0.88rem,8.2cqw,1.5rem)]'
          : 'text-[clamp(1.1rem,12cqw,2rem)]';
  return ['block max-w-full whitespace-normal break-keep hyphens-none text-center font-black leading-[1.16] tracking-[0.02em]', size].join(' ');
}

export function clampTotalCards(value: unknown): number {
  const parsed = typeof value === 'number' ? value : Number.parseInt(String(value ?? ''), 10);
  if (!Number.isFinite(parsed)) return defaultTotalCards;
  return Math.min(maxTotalCards, Math.max(minTotalCards, Math.round(parsed)));
}

export function autoNeutralCards(totalCards: number, startingTeamHandicap = 1): number {
  const total = clampTotalCards(totalCards);
  const handicap = Math.min(total, Math.max(0, Math.round(startingTeamHandicap)));
  const maxNeutral = total - handicap;
  let neutral = Math.round(total / 3);
  if (neutral > maxNeutral) neutral = maxNeutral;
  if ((total - neutral - handicap) % 2 !== 0) {
    if (neutral < maxNeutral) neutral += 1;
    else if (neutral > 0) neutral -= 1;
  }
  return neutral;
}

export function normalizeLobbySettingsForSave(settings: Settings): Settings {
  const totalCards = clampTotalCards(settings.totalCards ?? defaultTotalCards);
  const imageCardCount = Math.min(totalCards, Math.max(0, Math.round(settings.imageCardCount ?? 0)));
  const startingTeamHandicap = Math.min(totalCards, Math.max(0, Math.round(settings.startingTeamHandicap ?? 1)));
  if (settings.autoColorCounts !== false) {
    const neutralCards = autoNeutralCards(totalCards, startingTeamHandicap);
    return {
      ...settings,
      totalCards,
      autoColorCounts: true,
      blueCards: 0,
      redCards: 0,
      neutralCards,
      startingTeamHandicap,
      blackCards: Math.min(neutralCards, Math.max(0, Math.round(settings.blackCards ?? 0))),
      imageCardCount,
    };
  }

  let blueCards = Math.max(0, Math.round(settings.blueCards ?? 0));
  let redCards = Math.max(0, Math.round(settings.redCards ?? 0));
  const nonHandicapCards = totalCards - startingTeamHandicap;
  if (blueCards > nonHandicapCards) {
    blueCards = nonHandicapCards;
    redCards = 0;
  } else if (blueCards + redCards > nonHandicapCards) {
    redCards = nonHandicapCards - blueCards;
  }
  const neutralCards = Math.max(0, nonHandicapCards - blueCards - redCards);
  return {
    ...settings,
    totalCards,
    autoColorCounts: false,
    blueCards,
    redCards,
    neutralCards,
    startingTeamHandicap,
    blackCards: Math.min(neutralCards, Math.max(0, Math.round(settings.blackCards ?? 0))),
    imageCardCount,
  };
}

export function cardModeFromImageCount(imageCardCount: number, totalCards = defaultTotalCards): CardMode {
  if (imageCardCount <= 0) return 'words';
  if (imageCardCount >= clampTotalCards(totalCards)) return 'images';
  return 'mixed';
}

export function imageCountForMode(mode: CardMode, currentMixedCount: number, totalCards = defaultTotalCards): number {
  const total = clampTotalCards(totalCards);
  if (mode === 'words') return 0;
  if (mode === 'images') return total;
  return Math.min(total - 1, Math.max(1, currentMixedCount || 8));
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

export function shouldResetClueDraft(input: {
  reason: 'snapshot' | 'manual-reset';
  previousClue: ClueEntry | null | undefined;
  nextClue: ClueEntry | null | undefined;
  previousPhase?: GameplayPhase;
  nextPhase?: GameplayPhase;
}): boolean {
  if (input.reason === 'manual-reset') return true;
  if (input.previousPhase && input.nextPhase && input.previousPhase !== input.nextPhase && input.nextPhase === 'active') return true;
  const prev = input.previousClue;
  const next = input.nextClue;
  if (!prev && !next) return false;
  if (!prev || !next) return true;
  return prev.round !== next.round || prev.team !== next.team;
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
    else if (revealedStyle === 'greyed') styleClasses = 'opacity-30';
    else if (revealedStyle === 'invisible') styleClasses = 'invisible';
    else if (revealedStyle === 'omitted') styleClasses = 'hidden';
  }

  return {
    visibleColor,
    label: visibleColor === 'hidden' ? 'Unrevealed' : visibleColor[0].toUpperCase() + visibleColor.slice(1),
    isLastSelected,
    classes: [colorClass, styleClasses, showHiddenColor && !card.revealed ? 'shadow-inner shadow-white/10' : ''].filter(Boolean).join(' '),
  };
}


export function shouldCueChatMessage(viewer: Viewer | null | undefined, message: Pick<{ senderUserId: string }, 'senderUserId'>): boolean {
  const id = viewerId(viewer);
  return !id || message.senderUserId !== id;
}

export function chatCueNotice(message: Pick<{ displayName: string; body: string }, 'displayName' | 'body'>): string {
  const name = message.displayName?.trim() || 'Player';
  const compact = message.body.trim().replace(/\s+/g, ' ');
  const truncated = compact.length > 38 ? `${compact.slice(0, 38)}…` : compact;
  return `${name}: ${truncated}`;
}


export type EndGameOutcome = 'win' | 'loss' | 'neutral';

export interface MemoryCapturePlayer {
  name: string;
  spymaster: boolean;
  representative: boolean;
}

export interface MemoryCaptureTeam {
  key: 'blue' | 'red';
  name: string;
  color: string;
  players: MemoryCapturePlayer[];
}

export interface MemoryCaptureCard {
  badgeNumber: number;
  label: string;
  color: CardColor | 'hidden';
  contentType: 'word' | 'image';
  imageUrl?: string;
  isLastSelected: boolean;
}

export interface MemoryCaptureBoardLayout {
  boardColumnsMobile: number;
  boardColumnsDesktop: number;
  imageCardScale: ImageCardScale;
  strictCardAspectRatios: boolean;
}

export interface MemoryCaptureModel {
  roomId: string;
  title: string;
  subtitle: string;
  generatedLabel: string;
  roastLine: string;
  showNumberBadges: boolean;
  boardLayout: MemoryCaptureBoardLayout;
  winner: MemoryCaptureTeam;
  loser: MemoryCaptureTeam;
  cards: MemoryCaptureCard[];
}

export function endGameOutcome(winner: 'blue' | 'red' | '', viewer: Viewer | null | undefined, players: LobbyPlayer[]): EndGameOutcome {
  if (winner !== 'blue' && winner !== 'red') return 'neutral';
  const player = findViewerPlayer(players, viewer);
  if (!player || (player.team !== 'blue' && player.team !== 'red')) return 'neutral';
  return player.team === winner ? 'win' : 'loss';
}

export function buildMemoryCaptureModel(input: {
  roomId: string;
  winner: 'blue' | 'red';
  players: LobbyPlayer[];
  cards: GameplayCard[];
  settings: Settings;
  lastSelected?: LastSelected | null;
  showNumberBadges?: boolean;
  boardLayout?: Partial<MemoryCaptureBoardLayout>;
  roastTemplates?: string[];
  finishedAt?: string;
  generatedAt?: Date;
}): MemoryCaptureModel {
  const loser = input.winner === 'blue' ? 'red' : 'blue';
  const playersForTeam = (key: 'blue' | 'red', predicate: (player: LobbyPlayer) => boolean = () => true): LobbyPlayer[] => input.players
    .filter((player) => player.team === key && predicate(player) && player.displayName.trim());
  const playerNamesForTeam = (key: 'blue' | 'red', predicate: (player: LobbyPlayer) => boolean = () => true): string[] => playersForTeam(key, predicate)
    .map((player) => player.displayName.trim());
  const team = (key: 'blue' | 'red'): MemoryCaptureTeam => ({
    key,
    name: displayTeamName(key, input.settings),
    color: teamColor(key, input.settings),
    players: playersForTeam(key).map((player) => ({ name: player.displayName.trim(), spymaster: player.spymaster, representative: player.representative })),
  });
  const sorted = displayCards(input.cards, cardModeFromImageCount(input.settings.imageCardCount ?? 0, input.settings.totalCards ?? defaultTotalCards), input.settings.mixedImageOrderFirst);
  const generatedAt = input.finishedAt ? new Date(input.finishedAt) : (input.generatedAt ?? new Date());
  const generatedLabel = new Intl.DateTimeFormat('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
  }).format(generatedAt);
  const loserTeam = displayTeamName(loser, input.settings);
  const winnerTeam = displayTeamName(input.winner, input.settings);
  const roastLine = deterministicRoastLine({
    roomId: input.roomId,
    winnerTeam,
    loserTeam,
    winningPlayers: playerNamesForTeam(input.winner),
    losingPlayers: playerNamesForTeam(loser),
    winningSpymasters: playerNamesForTeam(input.winner, (player) => player.spymaster),
    losingSpymasters: playerNamesForTeam(loser, (player) => player.spymaster),
    winningGuessers: playerNamesForTeam(input.winner, (player) => !player.spymaster),
    losingGuessers: playerNamesForTeam(loser, (player) => !player.spymaster),
    templates: input.roastTemplates ?? [],
    disabled: Boolean(input.settings.memoryRoastsDisabled),
  });

  return {
    roomId: input.roomId,
    title: `${displayTeamName(input.winner, input.settings)} wins`,
    subtitle: '',
    generatedLabel,
    roastLine,
    showNumberBadges: input.showNumberBadges ?? true,
    boardLayout: {
      boardColumnsMobile: clampBoardColumns(input.boardLayout?.boardColumnsMobile, defaultGameplayPreferences.boardColumnsMobile),
      boardColumnsDesktop: clampBoardColumns(input.boardLayout?.boardColumnsDesktop, defaultGameplayPreferences.boardColumnsDesktop),
      imageCardScale: clampImageCardScale(input.boardLayout?.imageCardScale),
      strictCardAspectRatios: input.boardLayout?.strictCardAspectRatios ?? defaultGameplayPreferences.strictCardAspectRatios,
    },
    winner: team(input.winner),
    loser: team(loser),
    cards: sorted.map((card) => ({
      badgeNumber: card.badgeNumber,
      label: card.contentType === 'image' ? `Picture #${card.badgeNumber}` : toTitleCase(card.word) || `Card #${card.badgeNumber}`,
      color: card.revealed ? (card.color ?? 'hidden') : 'hidden',
      contentType: card.contentType,
      imageUrl: card.contentType === 'image' ? cardImageUrl(card) : undefined,
      isLastSelected: input.lastSelected?.index === card.originalIndex,
    })),
  };
}

export function deterministicRoastLine(input: {
  roomId: string;
  winnerTeam?: string;
  loserTeam: string;
  winningPlayers?: string[];
  losingPlayers?: string[];
  winningSpymasters?: string[];
  losingSpymasters?: string[];
  winningGuessers?: string[];
  losingGuessers?: string[];
  templates: string[];
  disabled: boolean;
}): string {
  if (input.disabled) return '';
  const templates = input.templates
    .map((line) => line.trim())
    .filter((line) => line && !line.startsWith('#'));
  if (!templates.length) return '';
  const seed = hashString(`${input.roomId}:${input.loserTeam}`);
  const template = templates[seed % templates.length];
  const winnerTeam = input.winnerTeam?.trim() || 'Winning team';
  const pick = (names: string[] | undefined, fallback: string): string => {
    const available = names?.map((name) => name.trim()).filter(Boolean) ?? [];
    return available.length ? available[seed % available.length] : fallback;
  };

  return template
    .replaceAll('{WINNER_TEAM}', winnerTeam)
    .replaceAll('{LOSER_TEAM}', input.loserTeam)
    .replaceAll('{RANDOM_WINNING_PLAYER}', pick(input.winningPlayers, winnerTeam))
    .replaceAll('{RANDOM_LOSING_PLAYER}', pick(input.losingPlayers, input.loserTeam))
    .replaceAll('{RANDOM_WINNING_SPYMASTER}', pick(input.winningSpymasters, winnerTeam))
    .replaceAll('{RANDOM_LOSING_SPYMASTER}', pick(input.losingSpymasters, input.loserTeam))
    .replaceAll('{RANDOM_WINNING_GUESSER}', pick(input.winningGuessers, winnerTeam))
    .replaceAll('{RANDOM_LOSING_GUESSER}', pick(input.losingGuessers, input.loserTeam));
}

export function hashString(value: string): number {
  let hash = 2166136261;
  for (let index = 0; index < value.length; index += 1) {
    hash ^= value.charCodeAt(index);
    hash = Math.imul(hash, 16777619);
  }
  return hash >>> 0;
}
