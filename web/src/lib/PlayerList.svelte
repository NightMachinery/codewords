<script lang="ts">
  import { playerBuckets, type LobbyPlayer, type Team } from './lobby';
  import type { Viewer } from './api';

  interface Props {
    players: LobbyPlayer[];
    viewer: Viewer | null;
    hostControls: boolean;
    roomHostId: string;
    onAssignTeam: (id: string, team: Team) => void;
    onToggleSpymaster: (id: string) => void;
    onToggleRepresentative: (id: string) => void;
    onToggleMod: (id: string) => void;
  }

  let {
    players,
    viewer,
    hostControls,
    roomHostId,
    onAssignTeam,
    onToggleSpymaster,
    onToggleRepresentative,
    onToggleMod
  }: Props = $props();

  let buckets = $derived(playerBuckets(players));
</script>

{#snippet roleBadges(player: LobbyPlayer)}
  {#if player.spymaster}
    <span class="rounded-full bg-slate-100 px-2.5 py-1 text-xs font-black text-slate-950">Spymaster</span>
  {/if}
  {#if player.representative}
    <span class="rounded-full bg-amber-200 px-2.5 py-1 text-xs font-black text-slate-950">Representative</span>
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
          {#if player.id === viewer?.playerId}
            <span class="ml-1 text-[10px] text-emerald-300 uppercase tracking-wider">(You)</span>
          {/if}
        </h3>
        <p class="text-xs text-slate-500">{player.id.slice(0, 8)}</p>
      </div>
      <div class="flex flex-wrap justify-end gap-2">{@render roleBadges(player)}</div>
    </div>
    <div class="mt-4 flex flex-wrap gap-2">
      {#if hostControls || player.id === viewer?.playerId}
        <button class={['rounded-full border px-3 py-1.5 text-xs font-bold transition', player.team === 'blue' ? 'border-blue-300 bg-blue-400/20 text-blue-100' : 'border-blue-300/50 text-blue-100/60 hover:bg-blue-400/20']} onclick={() => onAssignTeam(player.id, 'blue')}>Blue</button>
        <button class={['rounded-full border px-3 py-1.5 text-xs font-bold transition', player.team === 'red' ? 'border-red-300 bg-red-400/20 text-red-100' : 'border-red-300/50 text-red-100/60 hover:bg-red-400/20']} onclick={() => onAssignTeam(player.id, 'red')}>Red</button>
        <button class={['rounded-full border px-3 py-1.5 text-xs font-bold transition', player.team === 'observers' ? 'border-slate-300 bg-slate-400/20 text-slate-100' : 'border-slate-300/50 text-slate-100/60 hover:bg-slate-400/20']} onclick={() => onAssignTeam(player.id, 'observers')}>Observer</button>
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
    'border-slate-700 bg-slate-900/40']}>
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

<div class="space-y-6">
  <div class="grid grid-flow-dense gap-6 md:grid-cols-2">
    {@render TeamColumn('Blue team', 'blue', buckets.blue)}
    {@render TeamColumn('Red team', 'red', buckets.red)}
  </div>

  <div class="grid grid-flow-dense gap-6 md:grid-cols-2">
    {@render TeamColumn('Observers', 'observers', buckets.observers)}
    {@render TeamColumn('Unassigned', 'unassigned', buckets.unassigned)}
  </div>
</div>
