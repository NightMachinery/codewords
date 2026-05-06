<script lang="ts">
  import type { Settings } from './api';
  import { bottomShortcutItems, displayTeamName, formatClueNumber, hexWithAlpha, pressableButtonClasses, teamColor, type ClueEntry, type GameplayPhase } from './gameplay';
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
  <svg class="h-3.5 w-3.5" viewBox="0 0 24 24" aria-hidden="true">
    {#if kind === 'spy'}
      <path fill="currentColor" d="M4 9.5 6.2 5h11.6L20 9.5c-2.4.7-5.1 1-8 1s-5.6-.3-8-1Zm2.6 3.1c1.7.3 3.5.4 5.4.4s3.7-.1 5.4-.4l-.7 5.7c-.1 1-1 1.7-2 1.7H9.3c-1 0-1.9-.7-2-1.7l-.7-5.7Z" />
    {:else if kind === 'rep'}
      <path fill="currentColor" d="M12 3 4.5 6.4v5.1c0 4.7 3.2 8.1 7.5 9.5 4.3-1.4 7.5-4.8 7.5-9.5V6.4L12 3Z" />
    {:else if kind === 'board'}
      <path fill="currentColor" d="M4 4h7v7H4V4Zm9 0h7v7h-7V4ZM4 13h7v7H4v-7Zm9 0h7v7h-7v-7Z" />
    {:else if kind === 'players'}
      <path fill="currentColor" d="M8 11a4 4 0 1 1 0-8 4 4 0 0 1 0 8Zm8.5 1a3.5 3.5 0 1 1 0-7 3.5 3.5 0 0 1 0 7ZM2 20c.4-4 2.6-6 6-6s5.6 2 6 6H2Zm11.5 0a7.7 7.7 0 0 0-1.6-4.1c1-.6 2.2-.9 3.6-.9 3 0 5 1.7 5.5 5h-7.5Z" />
    {:else if kind === 'clues'}
      <path fill="currentColor" d="M5 4h14v3H5V4Zm0 5h10v3H5V9Zm0 5h14v3H5v-3Zm0 5h8v2H5v-2Z" />
    {:else if kind === 'settings'}
      <path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8.2a3.8 3.8 0 1 0 0 7.6 3.8 3.8 0 0 0 0-7.6Z" />
      <path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="1.8" d="m19.2 13.2 1.2 1.1-1.8 3.1-1.6-.5a7.3 7.3 0 0 1-1.6.9l-.3 1.7H8.9l-.3-1.7a7.3 7.3 0 0 1-1.6-.9l-1.6.5-1.8-3.1 1.2-1.1a7.5 7.5 0 0 1 0-2.4L3.6 9.7l1.8-3.1 1.6.5a7.3 7.3 0 0 1 1.6-.9l.3-1.7h6.2l.3 1.7a7.3 7.3 0 0 1 1.6.9l1.6-.5 1.8 3.1-1.2 1.1a7.5 7.5 0 0 1 0 2.4Z" />
    {:else if kind === 'local'}
      <path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="1.8" d="M5 6h14M5 12h14M5 18h14" />
      <circle cx="9" cy="6" r="2.1" fill="currentColor" />
      <circle cx="15" cy="12" r="2.1" fill="currentColor" />
      <circle cx="11" cy="18" r="2.1" fill="currentColor" />
    {:else}
      <path fill="currentColor" d="M4 5h16v11H8l-4 4V5Z" />
    {/if}
  </svg>
{/snippet}

{#if !controlsExpanded}
  <button
    class={pressableButtonClasses('fixed bottom-3 left-1/2 z-30 inline-flex -translate-x-1/2 items-center gap-2 rounded-full border border-slate-600/70 bg-slate-950/95 px-4 py-2 text-xs font-black uppercase tracking-[0.18em] text-slate-100 shadow-2xl backdrop-blur-md hover:border-emerald-300/70 hover:text-emerald-100')}
    onclick={() => (controlsExpanded = true)}
    aria-label="Expand bottom controls"
  >
    <span class="h-2.5 w-2.5 rounded-full" style={turnGlowStyle}></span>
    Controls
  </button>
{:else}
<footer class="fixed bottom-0 left-0 right-0 z-30 border-t border-slate-700/60 bg-slate-900/90 p-2 shadow-[0_-10px_30px_rgba(0,0,0,0.5)] backdrop-blur-md sm:p-3">
  <div class="relative mx-auto flex max-w-7xl flex-col gap-2 md:flex-row md:items-center md:justify-between md:gap-4">
    <button
      class={pressableButtonClasses('absolute -top-10 right-2 rounded-full border border-slate-700/80 bg-slate-950/95 px-3 py-1.5 text-[10px] font-black uppercase tracking-[0.18em] text-slate-300 shadow-xl hover:border-emerald-300/60 hover:text-emerald-100')}
      onclick={() => (controlsExpanded = false)}
      aria-label="Collapse bottom controls"
    >
      Collapse
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
                  <span class="max-w-28 truncate">{player.displayName.trim() || 'Player'}</span>
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
          <svg class="h-5 w-5" viewBox="0 0 24 24" aria-hidden="true">
            <path d="M4 9.2 6.1 5h11.8L20 9.2c-2.5.8-5.2 1.2-8 1.2s-5.5-.4-8-1.2Z" fill="currentColor" opacity="0.9" />
            <path d="M6.8 12.5c1.6.4 3.3.6 5.2.6s3.6-.2 5.2-.6l-.6 4.4A2.4 2.4 0 0 1 14.2 19H9.8a2.4 2.4 0 0 1-2.4-2.1l-.6-4.4Z" fill="currentColor" opacity="0.45" />
            <path d="M8.2 14.5h2.6M13.2 14.5h2.6" stroke="currentColor" stroke-linecap="round" stroke-width="1.8" />
            {#if spymasterViewActive}
              <circle cx="18" cy="6" r="2.4" fill="currentColor" />
            {/if}
          </svg>
        </button>
      {/if}
    </div>
  </div>
</footer>
{/if}
