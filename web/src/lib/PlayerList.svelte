<script lang="ts">
  import { playerBuckets, type LobbyPlayer, type Team } from './lobby';
  import type { Settings, Viewer } from './api';
  import { displayTeamName, hexWithAlpha, teamColor } from './gameplay';

  interface Props {
    players: LobbyPlayer[];
    viewer: Viewer | null;
    settings: Settings;
    hostControls: boolean;
    roomHostId: string;
    onAssignTeam: (id: string, team: Team) => void;
    onToggleSpymaster: (id: string) => void;
    onToggleRepresentative: (id: string) => void;
    onToggleMod: (id: string) => void;
    onRejoinTeam: (id: string) => void;
  }

  let {
    players,
    viewer,
    settings,
    hostControls,
    roomHostId,
    onAssignTeam,
    onToggleSpymaster,
    onToggleRepresentative,
    onToggleMod,
    onRejoinTeam
  }: Props = $props();

  let buckets = $derived(playerBuckets(players));
</script>

{#snippet SpyIcon()}
  <svg class="h-3.5 w-3.5" viewBox="0 0 24 24" aria-hidden="true">
    <path fill="currentColor" d="M4 9.5 6.2 5h11.6L20 9.5c-2.4.7-5.1 1-8 1s-5.6-.3-8-1Zm2.6 3.1c1.7.3 3.5.4 5.4.4s3.7-.1 5.4-.4l-.7 5.7c-.1 1-1 1.7-2 1.7H9.3c-1 0-1.9-.7-2-1.7l-.7-5.7ZM9 16.2c0 .8.7 1.5 1.5 1.5s1.5-.7 1.5-1.5H9Zm3 0c0 .8.7 1.5 1.5 1.5s1.5-.7 1.5-1.5h-3Z" />
  </svg>
{/snippet}

{#snippet RepIcon()}
  <svg class="h-3.5 w-3.5" viewBox="0 0 24 24" aria-hidden="true">
    <path fill="currentColor" d="M12 3 4.5 6.4v5.1c0 4.7 3.2 8.1 7.5 9.5 4.3-1.4 7.5-4.8 7.5-9.5V6.4L12 3Zm0 4.1 4 1.8v2.7c0 2.7-1.5 4.8-4 6-2.5-1.2-4-3.3-4-6V8.9l4-1.8Z" />
  </svg>
{/snippet}

{#snippet roleBadges(player: LobbyPlayer)}
  {#if player.spymaster}
    <span class="inline-flex items-center gap-1 rounded-full bg-slate-100 px-2 py-1 text-xs font-black text-slate-950">{@render SpyIcon()} Spy</span>
  {/if}
  {#if player.representative}
    <span class="inline-flex items-center gap-1 rounded-full bg-amber-200 px-2 py-1 text-xs font-black text-slate-950">{@render RepIcon()} Rep</span>
  {/if}
  {#if player.mod}
    <span class="rounded-full bg-emerald-200 px-2.5 py-1 text-xs font-black text-slate-950">Mod</span>
  {/if}
{/snippet}

{#snippet PlayerCard(player: LobbyPlayer)}
  <article class="group rounded-2xl border border-slate-700 bg-slate-950 p-4 transition duration-300 hover:-translate-y-0.5 hover:border-slate-500">
    <div class="flex items-start justify-between gap-3">
      <div>
        <h3 class="font-black text-slate-50">
          {player.displayName || 'Unnamed player'}
          {#if player.id === (viewer?.playerId || viewer?.userId)}
            <span class="ml-1 text-[10px] text-emerald-300 uppercase tracking-wider">(You)</span>
          {/if}
        </h3>
        <p class="text-xs text-slate-500">{player.id.slice(0, 8)}</p>
      </div>
      <div class="flex flex-wrap justify-end gap-2">{@render roleBadges(player)}</div>
    </div>
    <div class="mt-4 flex flex-wrap gap-2">
      {#if hostControls || player.id === (viewer?.playerId || viewer?.userId)}
        <button class={['rounded-full border px-3 py-1.5 text-xs font-bold transition', player.team === 'blue' ? 'text-white' : 'text-slate-100/70']} style={`border-color: ${hexWithAlpha(teamColor('blue', settings), player.team === 'blue' ? 'cc' : '80')}; background-color: ${player.team === 'blue' ? hexWithAlpha(teamColor('blue', settings), '40') : 'transparent'};`} onclick={() => onAssignTeam(player.id, 'blue')}>{displayTeamName('blue', settings)}</button>
        <button class={['rounded-full border px-3 py-1.5 text-xs font-bold transition', player.team === 'red' ? 'text-white' : 'text-slate-100/70']} style={`border-color: ${hexWithAlpha(teamColor('red', settings), player.team === 'red' ? 'cc' : '80')}; background-color: ${player.team === 'red' ? hexWithAlpha(teamColor('red', settings), '40') : 'transparent'};`} onclick={() => onAssignTeam(player.id, 'red')}>{displayTeamName('red', settings)}</button>
        <button class={['rounded-full border px-3 py-1.5 text-xs font-bold transition', player.team === 'observers' ? 'border-slate-300 bg-slate-400/20 text-slate-100' : 'border-slate-300/50 text-slate-100/60 hover:bg-slate-400/20']} onclick={() => onAssignTeam(player.id, 'observers')}>Observer</button>
      {/if}
      {#if player.team === 'observers' && player.previousTeam && (hostControls || player.id === (viewer?.playerId || viewer?.userId))}
        <button class="rounded-full border border-emerald-300/70 px-3 py-1.5 text-xs font-black text-emerald-100 hover:bg-emerald-300/10" onclick={() => onRejoinTeam(player.id)}>
          Rejoin {displayTeamName(player.previousTeam, settings)}
        </button>
      {/if}
      {#if hostControls && (player.team === 'blue' || player.team === 'red')}
        <button class={['rounded-full border px-3 py-1.5 text-xs font-bold transition', player.spymaster ? 'border-slate-100 bg-white text-slate-950' : 'border-slate-600 text-slate-200 hover:border-slate-300']} onclick={() => onToggleSpymaster(player.id)}>Spy</button>
        <button class={['rounded-full border px-3 py-1.5 text-xs font-bold transition', player.representative ? 'border-amber-200 bg-amber-200 text-slate-950' : 'border-slate-600 text-slate-200 hover:border-slate-300']} onclick={() => onToggleRepresentative(player.id)}>Rep</button>
      {/if}
      {#if hostControls && player.id !== roomHostId}
        <button class="rounded-full border border-emerald-400/60 px-3 py-1.5 text-xs font-bold text-emerald-100 hover:bg-emerald-400/15" onclick={() => onToggleMod(player.id)}>
          {player.mod ? 'Demote mod' : 'Promote mod'}
        </button>
      {/if}
    </div>
  </article>
{/snippet}

{#snippet TeamColumn(title: string, tone: 'blue' | 'red' | 'observers' | 'unassigned', members: LobbyPlayer[])}
  <section class={['rounded-[2rem] border p-5 shadow-2xl shadow-slate-950/25', 
    tone === 'blue' ? 'border-blue-300/30 bg-blue-400/10' : 
    tone === 'red' ? 'border-red-300/30 bg-red-400/10' : 
    tone === 'observers' ? 'border-slate-500/30 bg-slate-700/10' :
    'border-slate-700 bg-slate-900/40']}
    style={tone === 'blue' || tone === 'red' ? `border-color: ${hexWithAlpha(teamColor(tone, settings), '55')}; background-color: ${hexWithAlpha(teamColor(tone, settings), '18')};` : ''}>
    <h2 class="text-xl font-black tracking-tight">{title} ({members.length})</h2>
    <div class="mt-4 grid gap-3">
      {#each members as player (player.id)}
        {@render PlayerCard(player)}
      {:else}
        <p class="rounded-2xl border border-slate-700/70 bg-slate-950/70 px-4 py-6 text-center text-sm text-slate-400">Empty.</p>
      {/each}
    </div>
  </section>
{/snippet}

<div id="players" class="space-y-6">
  <div class="grid grid-flow-dense gap-6 md:grid-cols-2">
    {@render TeamColumn(displayTeamName('blue', settings), 'blue', buckets.blue)}
    {@render TeamColumn(displayTeamName('red', settings), 'red', buckets.red)}
  </div>

  <div class="grid grid-flow-dense gap-6 md:grid-cols-2">
    {@render TeamColumn('Observers', 'observers', buckets.observers)}
    {@render TeamColumn('Unassigned', 'unassigned', buckets.unassigned)}
  </div>
</div>
