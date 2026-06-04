<script lang="ts">
  import {
    boardGridContainerClasses,
    boardGridLayoutClasses,
    boardGridStyle,
    boardCardSpanClasses,
    cardAspectRatioClasses,
    cardChromeClasses,
    cardChromePaddingStyle,
    cardChromeStyle,
    cardContentLabel,
    cardDisabledStateClasses,
    cardImageUrl,
    cardViewState,
    cardWordTextClasses,
    cardWordTextSegments,
    fitCardWordShrinkPx,
    hexWithAlpha,
    imageCardGridStyle,
    imageColorFrameClasses,
    pressableButtonClasses,
    selectedImageOverlayStyle,
    teamColor,
    toTitleCase,
    type DisplayCard,
    type GameplayCard,
    type GameplayPhase,
    type GameplayPreferences,
    type LastSelected,
  } from './gameplay';
  import type { Settings } from './api';
  import { auroraPaletteFor, type ThemeId, type ThemeShaderSurface } from './theme';
  import AuroraBackground from './backgrounds/AuroraBackground.svelte';
  import FitCardWord from './FitCardWord.svelte';

  type BoardRole = {
    canSeeHiddenColors: boolean;
    kind: string;
    activeGuesser: boolean;
  };

  let {
    cards,
    settings,
    preferences,
    role,
    spymasterViewActive,
    lastSelected,
    phase,
    guessDisabledReason,
    onGuess,
    captureMode = false,
    theme = 'dark',
  } = $props<{
    cards: DisplayCard[];
    settings: Settings;
    preferences: Pick<GameplayPreferences, 'boardColumnsMobile' | 'boardColumnsDesktop' | 'imageCardScale' | 'strictCardAspectRatios' | 'showNumberBadges' | 'spymasterRevealedStyle'>;
    role: BoardRole;
    spymasterViewActive: boolean;
    lastSelected: LastSelected | null | undefined;
    phase: GameplayPhase;
    guessDisabledReason: (card?: GameplayCard) => string;
    onGuess?: (index: number, card: GameplayCard) => void;
    captureMode?: boolean;
    theme?: ThemeId;
  }>();

  function surfaceShaderTheme(theme: ThemeId, surface: ThemeShaderSurface, captureMode: boolean): ThemeId | null {
    if (captureMode) return null;
    return auroraPaletteFor(theme, surface).surfaceShaders?.[surface] ? theme : null;
  }

  let activeColumns = $derived(preferences.boardColumnsDesktop);
  let mobileColumns = $derived(preferences.boardColumnsMobile);
  let boardShaderTheme = $derived(surfaceShaderTheme(theme, 'board', captureMode));
  let cardShaderTheme = $derived(surfaceShaderTheme(theme, 'card', captureMode));
  let surfaceShadersActive = $derived(Boolean(boardShaderTheme || cardShaderTheme));
</script>

<div class={[boardGridContainerClasses(), 'board-grid-shell', surfaceShadersActive && 'surface-shader-board-grid-shell'].filter(Boolean).join(' ')}>
  {#if boardShaderTheme}
    <div class="surface-board-shader">
      <AuroraBackground theme={boardShaderTheme} surface="board" intensity={0.78} speed={0.34} />
    </div>
  {/if}
  <div id={captureMode ? undefined : 'board'} class={[boardGridLayoutClasses(captureMode), 'relative z-10'].join(' ')} style={boardGridStyle(mobileColumns, activeColumns)}>
    {#each cards as card (`${card.word ?? card.imageId ?? 'card'}-${card.originalIndex}`)}
      {@const showHiddenColor = role.canSeeHiddenColors && (role.kind !== 'spymaster' || spymasterViewActive)}
      {@const revealedStyle = (role.kind === 'spymaster' && spymasterViewActive) ? preferences.spymasterRevealedStyle : 'normal'}
      {@const view = cardViewState(card, card.originalIndex, showHiddenColor, lastSelected, revealedStyle)}
      {@const customColor = card.color === 'blue' ? teamColor('blue', settings) : card.color === 'red' ? teamColor('red', settings) : card.color === 'unity' ? teamColor('unity', settings) : ''}
      {@const disabledReason = guessDisabledReason(card)}
      <button
        class={pressableButtonClasses(['group relative', boardCardSpanClasses(captureMode), 'rounded-xl border text-left duration-200 hover:-translate-y-0.5 disabled:cursor-not-allowed disabled:hover:translate-y-0', surfaceShadersActive ? 'surface-shader-board-card' : '', cardAspectRatioClasses(card, preferences.strictCardAspectRatios), cardChromeClasses(card, view.isLastSelected), view.classes, cardDisabledStateClasses({ disabled: !role.activeGuesser || card.revealed || phase !== 'active', revealed: card.revealed, revealedStyle })].join(' '))}
        style={`${imageCardGridStyle(card, activeColumns, preferences.imageCardScale, mobileColumns)} ${cardChromePaddingStyle(card)} ${cardChromeStyle(card, view.visibleColor, customColor, view.isLastSelected)}`}
        disabled={Boolean(disabledReason)}
        title={disabledReason || `Reveal ${cardContentLabel(card)}`}
        onclick={() => onGuess?.(card.originalIndex, card)}
      >
        {#if preferences.showNumberBadges}
          <span class="absolute left-0 top-0 z-10 rounded-br-xl bg-slate-950/85 px-1.5 py-1 text-[10px] font-black leading-none text-slate-100">
            #{card.badgeNumber}
          </span>
        {/if}
        {#if card.contentType === 'image'}
          <img class="h-full w-full rounded-lg object-cover" src={cardImageUrl(card)} alt="Card illustration" loading={captureMode ? 'eager' : 'lazy'} />
          {#if view.visibleColor !== 'hidden' && customColor && view.isLastSelected}
            <span class={imageColorFrameClasses(view.isLastSelected)} style={`border-color: ${hexWithAlpha(customColor, 'E6')};`}></span>
          {/if}
          {#if view.isLastSelected}
            <span class="pointer-events-none absolute inset-0 z-30 rounded-xl border-4" style={selectedImageOverlayStyle(view.visibleColor, customColor)}></span>
          {/if}
        {:else}
          {@const wordSegments = cardWordTextSegments(toTitleCase(card.word) || 'Card')}
          <FitCardWord segments={wordSegments} classes={cardWordTextClasses(card.word)} shrinkPx={fitCardWordShrinkPx(captureMode)} />
          {#if view.isLastSelected}
            <span class="pointer-events-none absolute inset-0 z-30 rounded-xl border-4 border-emerald-200"></span>
          {/if}
        {/if}
      </button>
    {:else}
      <p class="col-span-full rounded-2xl border border-slate-700 bg-slate-950 p-6 text-slate-300">Waiting for the board snapshot...</p>
    {/each}
  </div>
  {#if cardShaderTheme}
    <div class="surface-card-shader">
      <AuroraBackground theme={cardShaderTheme} surface="card" intensity={0.64} speed={0.58} />
    </div>
  {/if}
</div>

<style>
  .board-grid-shell {
    position: relative;
    isolation: isolate;
  }

  .surface-shader-board-grid-shell {
    border-radius: 1.25rem;
  }

  .surface-board-shader {
    position: absolute;
    inset: -0.75rem;
    z-index: 0;
    overflow: hidden;
    border-radius: 1.5rem;
    opacity: 0.82;
    filter: saturate(1.08);
    pointer-events: none;
  }

  .surface-card-shader {
    position: absolute;
    inset: 0;
    z-index: 20;
    overflow: hidden;
    border-radius: 1.15rem;
    opacity: 0.28;
    mix-blend-mode: screen;
    pointer-events: none;
  }

  :global([data-theme='dracula']) .surface-shader-board-card {
    background-color: oklch(19% 0.035 294 / 0.74);
    border-color: oklch(74% 0.16 306 / 0.2);
    box-shadow:
      inset 0 0 0 1px oklch(100% 0 0 / 0.035),
      0 14px 28px oklch(8% 0.03 294 / 0.26);
  }

  :global([data-theme='dracula']) .surface-shader-board-card:hover:not(:disabled) {
    border-color: oklch(70% 0.18 342 / 0.38);
    box-shadow:
      inset 0 0 0 1px oklch(74% 0.16 306 / 0.16),
      0 16px 34px oklch(8% 0.03 294 / 0.34);
  }

  :global([data-theme='dracula']) .surface-shader-board-card.revealed-civilian-card {
    background-color: oklch(31% 0.075 76 / 0.9);
    border-color: oklch(83% 0.13 82 / 0.72);
    box-shadow:
      inset 0 0 0 1px oklch(92% 0.1 84 / 0.22),
      inset 0 0 24px oklch(76% 0.12 78 / 0.12),
      0 14px 28px oklch(8% 0.03 294 / 0.28);
  }

  :global([data-theme='glitch']) .surface-shader-board-card {
    background-color: oklch(13% 0.035 250 / 0.78);
    border-color: oklch(82% 0.18 210 / 0.18);
    box-shadow:
      inset 0 0 0 1px oklch(88% 0.22 145 / 0.045),
      0 14px 28px oklch(4% 0.02 250 / 0.36);
  }

  :global([data-theme='glitch']) .surface-shader-board-card:hover:not(:disabled) {
    border-color: oklch(84% 0.2 210 / 0.34);
    box-shadow:
      inset 0 0 0 1px oklch(88% 0.22 145 / 0.14),
      0 16px 34px oklch(3% 0.02 250 / 0.46);
  }
</style>
