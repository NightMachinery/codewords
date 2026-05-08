<script lang="ts">
  import {
    boardGridContainerClasses,
    boardGridLayoutClasses,
    boardGridStyle,
    boardCardSpanClasses,
    cardAspectRatioClasses,
    cardChromeClasses,
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
  }>();

  let activeColumns = $derived(preferences.boardColumnsDesktop);
  let mobileColumns = $derived(preferences.boardColumnsMobile);
</script>

<div class={boardGridContainerClasses()}>
  <div id={captureMode ? undefined : 'board'} class={boardGridLayoutClasses(captureMode)} style={boardGridStyle(mobileColumns, activeColumns)}>
    {#each cards as card (`${card.word ?? card.imageId ?? 'card'}-${card.originalIndex}`)}
      {@const showHiddenColor = role.canSeeHiddenColors && (role.kind !== 'spymaster' || spymasterViewActive)}
      {@const revealedStyle = (role.kind === 'spymaster' && spymasterViewActive) ? preferences.spymasterRevealedStyle : 'normal'}
      {@const view = cardViewState(card, card.originalIndex, showHiddenColor, lastSelected, revealedStyle)}
      {@const customColor = card.color === 'blue' ? teamColor('blue', settings) : card.color === 'red' ? teamColor('red', settings) : ''}
      {@const disabledReason = guessDisabledReason(card)}
      <button
        class={pressableButtonClasses(['group relative', boardCardSpanClasses(captureMode), 'rounded-xl border text-left duration-200 hover:-translate-y-0.5 disabled:cursor-not-allowed disabled:hover:translate-y-0', cardAspectRatioClasses(card, preferences.strictCardAspectRatios), cardChromeClasses(card, view.isLastSelected), view.classes, cardDisabledStateClasses({ disabled: !role.activeGuesser || card.revealed || phase !== 'active', revealed: card.revealed, revealedStyle })].join(' '))}
        style={`${imageCardGridStyle(card, activeColumns, preferences.imageCardScale, mobileColumns)} ${cardChromeStyle(card, view.visibleColor, customColor, view.isLastSelected)}`}
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
</div>
