<script lang="ts">
  import type { Settings } from './api';
  import { displayTeamName, formatClueNumber, hexWithAlpha, teamColor, type ClueEntry, type GameplayPhase } from './gameplay';
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

  let isYourTeam = $derived(role.team === currentTeam);
  let turnLabel = $derived(isYourTeam ? 'Your Turn' : 'Their Turn');
  let teamLabel = $derived(displayTeamName(currentTeam, settings));
  let ownTeamPlayers = $derived(players.filter((player) => player.team === role.team && (role.team === 'blue' || role.team === 'red')));
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
      <path fill="currentColor" d="M12 8a4 4 0 1 1 0 8 4 4 0 0 1 0-8Zm8.8 3 .2 1-.2 1-2.1.7-.6 1.4 1 2-1.4 1.4-2-1-1.4.6-.7 2.1-1 .2-1-.2-.7-2.1-1.4-.6-2 1-1.4-1.4 1-2-.6-1.4-2.1-.7-.2-1 .2-1 2.1-.7.6-1.4-1-2 1.4-1.4 2 1 1.4-.6.7-2.1 1-.2 1 .2.7 2.1 1.4.6 2-1 1.4 1.4-1 2 .6 1.4 2.1.7Z" />
    {:else if kind === 'local'}
      <path fill="currentColor" d="M4 5h16v4H4V5Zm0 6h10v4H4v-4Zm0 6h16v2H4v-2Zm12-6h4v4h-4v-4Z" />
    {:else}
      <path fill="currentColor" d="M4 5h16v11H8l-4 4V5Z" />
    {/if}
  </svg>
{/snippet}

<footer class="fixed bottom-0 left-0 right-0 z-30 border-t border-slate-700/60 bg-slate-900/90 p-3 backdrop-blur-md shadow-[0_-10px_30px_rgba(0,0,0,0.5)]">
  <div class="mx-auto max-w-7xl flex flex-wrap items-center justify-between gap-4">
    <!-- Turn Info -->
    <div class="flex items-center gap-4">
      <div class={['rounded-2xl px-4 py-2 border transition', 
        currentTeam === 'blue' ? 'border-blue-300/40 bg-blue-500/20 text-blue-100' : 
        currentTeam === 'red' ? 'border-red-300/40 bg-red-500/20 text-red-100' : 
        'border-slate-700 bg-slate-800 text-slate-400']}
        style={currentTeam === 'blue' || currentTeam === 'red' ? `border-color: ${hexWithAlpha(teamColor(currentTeam, settings), '80')}; background-color: ${hexWithAlpha(teamColor(currentTeam, settings), '2b')}; color: ${teamColor(currentTeam, settings)};` : ''}>
        <p class="text-[10px] font-black uppercase tracking-widest opacity-70">Turn</p>
        <h2 class="text-lg font-black tracking-tight">{turnLabel}</h2>
        <p class="text-xs font-bold opacity-75">{teamLabel}</p>
      </div>

      {#if currentClue}
        <div class="rounded-2xl border border-slate-700 bg-slate-950/50 px-4 py-2">
          <p class="text-[10px] font-black uppercase tracking-widest text-slate-500">Active Clue</p>
          <p class="font-black text-slate-100">{currentClue.text} · {formatClueNumber(currentClue.number)}</p>
        </div>
      {/if}
      {#if ownTeamPlayers.length}
        <div class="max-w-64 rounded-2xl border border-slate-700 bg-slate-950/50 px-3 py-2">
          <p class="text-[10px] font-black uppercase tracking-widest text-slate-500">{displayTeamName(role.team ?? '', settings)}</p>
          <div class="mt-1 flex flex-wrap gap-1.5">
            {#each ownTeamPlayers as player (player.id)}
              <span class="inline-flex max-w-28 items-center gap-1 truncate rounded-full bg-slate-800 px-2 py-1 text-[11px] font-bold text-slate-200">
                <span class="truncate">{player.displayName || 'Player'}</span>
                {#if player.spymaster}{@render MiniIcon('spy')}{/if}
                {#if player.representative}{@render MiniIcon('rep')}{/if}
              </span>
            {/each}
          </div>
        </div>
      {/if}
    </div>

    <!-- Controls -->
    <div class="flex flex-1 items-center justify-center gap-3 max-w-2xl">
      {#if role.kind === 'spymaster' && phase === 'active' && cluePermission.allowed}
        <div class="flex w-full gap-2">
          <input
            class="flex-1 rounded-xl border border-slate-700 bg-slate-950 px-3 py-2 text-sm font-semibold text-slate-50 outline-none ring-emerald-300 transition focus:ring-2 disabled:opacity-50"
            bind:value={clueText}
            maxlength="40"
            placeholder="One-word clue"
            disabled={!cluePermission.allowed}
          />
          <select class="w-24 rounded-xl border border-slate-700 bg-slate-950 px-2 py-2 text-sm text-slate-50" bind:value={clueNumber} disabled={!cluePermission.allowed}>
            <option value="">#</option>
            {#each [1, 2, 3, 4, 5, 6, 7, 8, 9] as n (n)}
              <option value={String(n)}>{n}</option>
            {/each}
            {#if settings.allowInfinityClue}
              <option value="∞">∞</option>
            {/if}
          </select>
          <button 
            class="rounded-xl bg-emerald-300 px-4 py-2 text-sm font-black text-slate-950 transition hover:bg-emerald-200 disabled:opacity-50" 
            disabled={Boolean(clueProblem) || !cluePermission.allowed} 
            onclick={onSubmitClue}
          >
            Submit
          </button>
        </div>
      {:else if role.activeGuesser && phase === 'active'}
        <div class="flex items-center gap-4 w-full">
          <p class="text-xs font-bold text-slate-400 flex-1 text-center">{guessProblem || 'Select a card to guess'}</p>
          <button 
            class="rounded-xl border border-slate-500 px-6 py-2 text-sm font-black text-slate-100 transition hover:border-emerald-300 hover:text-emerald-200 disabled:opacity-50" 
            disabled={Boolean(passProblem)} 
            onclick={onPassTurn}
          >
            Pass turn
          </button>
        </div>
      {:else if controlMessage}
        <div class="w-full rounded-2xl border border-slate-700 bg-slate-950/60 px-4 py-3 text-center">
          <p class="text-sm font-bold text-slate-300">{controlMessage}</p>
        </div>
      {/if}
    </div>

    <!-- Actions -->
    <div class="flex items-center gap-2">
      <div class="flex items-center gap-1 rounded-2xl border border-slate-700 bg-slate-950/50 p-1">
        {#each [
          ['board', 'board', 'Board'],
          ['players', 'players', 'Players'],
          ['clues', 'clues', 'Clues'],
          ['settings', 'settings', 'Settings'],
          ['local', 'local-options', 'Local'],
          ['chat', 'chat', 'Chat']
        ] as shortcut (shortcut[1])}
          <button class="grid h-8 w-8 place-items-center rounded-xl text-slate-300 transition hover:bg-slate-800 hover:text-emerald-200" title={shortcut[2]} onclick={() => onNavigate(shortcut[1])}>
            {@render MiniIcon(shortcut[0] as any)}
          </button>
        {/each}
      </div>
      {#if role.kind === 'spymaster'}
        <button 
          class={['rounded-xl border px-3 py-2 text-xs font-bold transition', spymasterViewActive ? 'border-emerald-300/50 bg-emerald-400/10 text-emerald-200' : 'border-slate-600 bg-slate-800 text-slate-300']} 
          onclick={onToggleView}
        >
          {spymasterViewActive ? 'Spy View: ON' : 'Spy View: OFF'}
        </button>
      {/if}
    </div>
  </div>
</footer>
