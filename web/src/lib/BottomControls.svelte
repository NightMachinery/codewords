<script lang="ts">
  import type { Settings } from './api';
  import { bottomShortcutItems, displayTeamName, formatClueNumber, hexWithAlpha, pressableButtonClasses, teamColor, type ClueEntry, type GameplayPhase } from './gameplay';
  import { Grid2X2, List, MessageSquare, Settings as SettingsIcon, SlidersHorizontal, Users, ChevronDown } from 'lucide-svelte';
  import { customSvg } from './customSvg';
  import SvgMaskIcon from './SvgMaskIcon.svelte';
  import type { LobbyPlayer, Team } from './lobby';

  interface Props {
    phase: GameplayPhase;
    currentTeam: Team;
    currentClue: ClueEntry | null;
    role: { kind: string; activeGuesser: boolean; team?: Team; player?: LobbyPlayer };
    cluePermission: { allowed: boolean; reason: string };
    clueText: string;
    clueNumber: string;
    clueProblem: string;
    guessProblem: string;
    passProblem: string;
    activeTeamHasRepresentative: boolean;
    settings: Settings;
    players: LobbyPlayer[];
    spymasterViewActive: boolean;
    onToggleView: () => void;
    onNavigate: (target: string) => void;
    onSubmitClue: () => void;
    onPassTurn: () => void;
  }

  let {
    phase,
    currentTeam,
    currentClue,
    role,
    cluePermission,
    clueText = $bindable(),
    clueNumber = $bindable(),
    clueProblem,
    guessProblem,
    passProblem,
    activeTeamHasRepresentative,
    settings,
    players,
    spymasterViewActive,
    onToggleView,
    onNavigate,
    onSubmitClue,
    onPassTurn
  }: Props = $props();

  let controlsExpanded = $state(true);

  function setControlsExpanded(expanded: boolean) {
    controlsExpanded = expanded;
    window.requestAnimationFrame(() => window.dispatchEvent(new CustomEvent('codewords:layout-change')));
  }
  let isYourTeam = $derived(role.team === currentTeam);
  let teamLabel = $derived(displayTeamName(currentTeam, settings));
  let currentTeamPlayers = $derived(players.filter((player) => player.team === currentTeam && (currentTeam === 'blue' || currentTeam === 'red')));
  let canActNow = $derived(Boolean(phase === 'active' && isYourTeam && (role.activeGuesser || (role.kind === 'spymaster' && cluePermission.allowed))));
  let turnColor = $derived(teamColor(currentTeam, settings));
  let turnGlowStyle = $derived(currentTeam === 'blue' || currentTeam === 'red'
    ? `background-color: ${turnColor}; box-shadow: 0 0 0 1px ${hexWithAlpha(turnColor, '88')}, 0 0 ${canActNow ? '34px' : '18px'} ${hexWithAlpha(turnColor, canActNow ? 'AA' : '66')};`
    : '');
  let controlMessage = $derived.by(() => {
    if (phase !== 'active') return 'Waiting for the match to start.';
    if (!role.player) return 'Spectators are read-only.';
    if (!isYourTeam) return 'Their turn. Watch the board.';
    if (role.kind === 'spymaster' && !cluePermission.allowed) return 'Your team is guessing. Watch the board.';
    if (!role.activeGuesser && role.kind !== 'spymaster') return activeTeamHasRepresentative ? 'Your representative will play for you.' : 'Your teammate will guess for you.';
    return '';
  });
</script>

{#snippet MiniIcon(kind: 'spy' | 'rep' | 'board' | 'players' | 'clues' | 'settings' | 'local' | 'chat')}
  {#if kind === 'spy'}
    <SvgMaskIcon src={customSvg.spy} classes="h-3.5 w-3.5" />
  {:else if kind === 'rep'}
    <SvgMaskIcon src={customSvg.representative} classes="h-3.5 w-3.5" />
  {:else if kind === 'board'}
    <Grid2X2 class="h-3.5 w-3.5" />
  {:else if kind === 'players'}
    <Users class="h-3.5 w-3.5" />
  {:else if kind === 'clues'}
    <List class="h-3.5 w-3.5" />
  {:else if kind === 'settings'}
    <SettingsIcon class="h-3.5 w-3.5" />
  {:else if kind === 'local'}
    <SlidersHorizontal class="h-3.5 w-3.5" />
  {:else}
    <MessageSquare class="h-3.5 w-3.5" />
  {/if}
{/snippet}

{#if !controlsExpanded}
  <button
    class={pressableButtonClasses('fixed bottom-3 left-1/2 z-30 inline-flex -translate-x-1/2 items-center gap-2 rounded-full border border-slate-600/70 bg-slate-950/95 px-4 py-2 text-xs font-black uppercase tracking-[0.18em] text-slate-100 shadow-2xl backdrop-blur-md hover:border-emerald-300/70 hover:text-emerald-100')}
    onclick={() => setControlsExpanded(true)}
    aria-label="Expand bottom controls"
  >
    <span class="h-2.5 w-2.5 rounded-full" style={turnGlowStyle}></span>
    Controls
  </button>
{:else}
<footer id="bottom-sticky-panel" class="fixed bottom-0 left-0 right-0 z-30 border-t border-slate-700/60 bg-slate-900/90 p-2 shadow-[0_-10px_30px_rgba(0,0,0,0.5)] backdrop-blur-md sm:p-3">
  <div class="relative mx-auto flex max-w-7xl flex-col gap-2 pt-9 md:flex-row md:items-center md:justify-between md:gap-4 md:pt-0 md:pr-12">
    <button
      class={pressableButtonClasses('absolute right-0 top-0 grid h-9 w-9 place-items-center rounded-bl-2xl border-b border-l border-slate-700/80 bg-slate-950/95 text-slate-300 shadow-xl hover:border-emerald-300/60 hover:text-emerald-100')}
      onclick={() => setControlsExpanded(false)}
      aria-label="Collapse bottom controls"
    >
      <ChevronDown class="h-5 w-5" />
    </button>

    <!-- Turn/team row -->
    <div class="relative flex min-w-0 flex-1 flex-col gap-2 md:max-w-sm">
      <div class="relative isolate min-h-10 overflow-hidden rounded-2xl border border-slate-700/70 bg-slate-950/55 px-3 py-2">
        {#if currentTeam === 'blue' || currentTeam === 'red'}
          <span class="pointer-events-none absolute -left-5 top-1/2 h-20 w-20 -translate-y-1/2 rounded-full opacity-35 blur-xl" style={`background-color: ${turnColor};`}></span>
        {/if}
        <div class="relative z-10 flex min-w-0 items-center gap-3">
          <span
            class={['h-4 w-4 shrink-0 rounded-full', canActNow ? 'animate-pulse' : ''].join(' ')}
            style={turnGlowStyle}
            title={`${teamLabel} turn`}
            aria-label={`${teamLabel} turn${canActNow ? ', you can act now' : ''}`}
          ></span>
          <div class="min-w-0 flex-1">
            <p class="truncate text-xs font-black uppercase tracking-[0.18em]" style={currentTeam === 'blue' || currentTeam === 'red' ? `color: ${turnColor};` : ''}>{teamLabel}</p>
            {#if currentTeamPlayers.length}
              <div class="mt-0.5 flex min-w-0 flex-wrap gap-x-3 gap-y-1 text-xs font-bold text-slate-200">
                {#each currentTeamPlayers as player (player.id)}
                  <span class={[
                    'inline-flex max-w-32 items-center gap-1 truncate',
                    player.spymaster ? 'bg-gradient-to-r from-slate-50 via-cyan-100 to-emerald-100 bg-clip-text text-transparent drop-shadow-[0_0_10px_rgba(125,211,252,0.35)]' : '',
                    player.representative ? 'bg-gradient-to-r from-amber-100 via-orange-100 to-rose-100 bg-clip-text text-transparent drop-shadow-[0_0_10px_rgba(251,191,36,0.35)]' : ''
                  ].filter(Boolean).join(' ')}>
                    {#if player.spymaster}<SvgMaskIcon src={customSvg.spy} classes="h-3.5 w-3.5 text-cyan-100" />{/if}
                    {#if player.representative}<SvgMaskIcon src={customSvg.representative} classes="h-3.5 w-3.5 text-amber-100" />{/if}
                    {player.displayName.trim() || 'Player'}
                  </span>
                {/each}
              </div>
            {:else}
              <p class="text-xs font-bold text-slate-500">Waiting for team.</p>
            {/if}
          </div>
        </div>
      </div>

      {#if currentClue}
        <div class="rounded-2xl border border-slate-700 bg-slate-950/50 px-3 py-2">
          <p class="text-[10px] font-black uppercase tracking-widest text-slate-500">Active Clue</p>
          <p class="truncate font-black text-slate-100">{currentClue.text} · {formatClueNumber(currentClue.number)}</p>
        </div>
      {/if}
    </div>

    <!-- Controls -->
    <div class="flex min-w-0 flex-1 items-center justify-center gap-3 md:max-w-2xl">
      {#if role.kind === 'spymaster' && phase === 'active' && cluePermission.allowed}
        <div class="flex w-full gap-2">
          <input
            class="min-w-0 flex-1 rounded-xl border border-slate-700 bg-slate-950 px-3 py-2 text-sm font-semibold text-slate-50 outline-none ring-emerald-300 transition focus:ring-2 disabled:opacity-50"
            bind:value={clueText}
            maxlength="40"
            placeholder="One-word clue"
            disabled={!cluePermission.allowed}
          />
          <select class="w-20 rounded-xl border border-slate-700 bg-slate-950 px-2 py-2 text-sm text-slate-50 sm:w-24" bind:value={clueNumber} disabled={!cluePermission.allowed}>
            <option value="">#</option>
            {#each [1, 2, 3, 4, 5, 6, 7, 8, 9] as n (n)}
              <option value={String(n)}>{n}</option>
            {/each}
            {#if settings.allowInfinityClue}
              <option value="∞">∞</option>
            {/if}
          </select>
          <button
            class={pressableButtonClasses('rounded-xl bg-emerald-300 px-4 py-2 text-sm font-black text-slate-950 hover:bg-emerald-200 disabled:opacity-50')}
            disabled={Boolean(clueProblem) || !cluePermission.allowed}
            onclick={onSubmitClue}
          >
            Submit
          </button>
        </div>
      {:else if role.activeGuesser && phase === 'active'}
        <div class="flex w-full items-center gap-3">
          <p class="min-w-0 flex-1 text-center text-xs font-bold text-slate-400">{guessProblem || 'Select a card to guess'}</p>
          <button
            class={pressableButtonClasses('rounded-xl border border-slate-500 px-5 py-2 text-sm font-black text-slate-100 hover:border-emerald-300 hover:text-emerald-200 disabled:opacity-50')}
            disabled={Boolean(passProblem)}
            onclick={onPassTurn}
          >
            Pass
          </button>
        </div>
      {:else if controlMessage}
        <div class="w-full rounded-2xl border border-slate-700 bg-slate-950/60 px-4 py-3 text-center">
          <p class="text-sm font-bold text-slate-300">{controlMessage}</p>
        </div>
      {/if}
    </div>

    <!-- Actions -->
    <div class="flex items-center justify-end gap-2">
      <div class="flex items-center gap-1 rounded-2xl border border-slate-700 bg-slate-950/50 p-1">
        {#each bottomShortcutItems as shortcut (shortcut.target)}
          <button class={pressableButtonClasses('grid h-8 w-8 place-items-center rounded-xl text-slate-300 hover:bg-slate-800 hover:text-emerald-200')} title={shortcut.label} aria-label={shortcut.label} onclick={() => onNavigate(shortcut.target)}>
            {@render MiniIcon(shortcut.kind)}
          </button>
        {/each}
      </div>
      {#if role.kind === 'spymaster'}
        <button
          class={pressableButtonClasses(['grid h-10 w-10 place-items-center rounded-xl border', spymasterViewActive ? 'border-emerald-300/60 bg-emerald-300/15 text-emerald-100' : 'border-slate-600 bg-slate-800 text-slate-300 hover:border-emerald-300/50'].join(' '))}
          onclick={onToggleView}
          aria-label={spymasterViewActive ? 'Turn spy view off' : 'Turn spy view on'}
          aria-pressed={spymasterViewActive}
          title={spymasterViewActive ? 'Spy view on' : 'Spy view off'}
        >
<span class="relative"><SvgMaskIcon src={customSvg.spy} classes="h-5 w-5" />{#if spymasterViewActive}<span class="absolute -right-1 -top-1 h-2.5 w-2.5 rounded-full bg-current"></span>{/if}</span>
        </button>
      {/if}
    </div>
  </div>
</footer>
{/if}
