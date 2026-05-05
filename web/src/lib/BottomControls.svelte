<script lang="ts">
  import type { Settings } from './api';
  import { formatClueNumber, type ClueEntry, type GameplayPhase } from './gameplay';
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
    spymasterViewActive: boolean;
    onToggleView: () => void;
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
    spymasterViewActive,
    onToggleView,
    onSubmitClue,
    onPassTurn
  }: Props = $props();

  let isYourTeam = $derived(role.team === currentTeam);
  let turnLabel = $derived(isYourTeam ? 'Your Turn' : 'Their Turn');
  let teamLabel = $derived(currentTeam === 'blue' ? 'Blue team' : currentTeam === 'red' ? 'Red team' : 'Waiting');
  let controlMessage = $derived.by(() => {
    if (phase !== 'active') return 'Waiting for the match to start.';
    if (!role.player) return 'Spectators are read-only.';
    if (!isYourTeam) return 'Their turn. Watch the board.';
    if (role.kind === 'spymaster' && !cluePermission.allowed) return 'Your team is guessing. Watch the board.';
    if (!role.activeGuesser && role.kind !== 'spymaster') return activeTeamHasRepresentative ? 'Your representative will play for you.' : 'Your teammate will guess for you.';
    return '';
  });
</script>

<footer class="fixed bottom-0 left-0 right-0 z-30 border-t border-slate-700/60 bg-slate-900/90 p-3 backdrop-blur-md shadow-[0_-10px_30px_rgba(0,0,0,0.5)]">
  <div class="mx-auto max-w-7xl flex flex-wrap items-center justify-between gap-4">
    <!-- Turn Info -->
    <div class="flex items-center gap-4">
      <div class={['rounded-2xl px-4 py-2 border transition', 
        currentTeam === 'blue' ? 'border-blue-300/40 bg-blue-500/20 text-blue-100' : 
        currentTeam === 'red' ? 'border-red-300/40 bg-red-500/20 text-red-100' : 
        'border-slate-700 bg-slate-800 text-slate-400']}>
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
