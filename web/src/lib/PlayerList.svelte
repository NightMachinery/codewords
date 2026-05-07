<script lang="ts">
  import { visiblePlayerBuckets, type LobbyPlayer, type Team } from './lobby';
  import type { Settings, Viewer } from './api';
  import { displayTeamName, hexWithAlpha, teamColor } from './gameplay';
  import { customSvg } from './customSvg';
  import SvgMaskIcon from './SvgMaskIcon.svelte';

  interface Props {
    players: LobbyPlayer[];
    viewer: Viewer | null;
    settings: Settings;
    hostControls: boolean;
    phase?: 'lobby' | 'active' | 'game_over';
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
    phase = 'lobby',
    roomHostId,
    onAssignTeam,
    onToggleSpymaster,
    onToggleRepresentative,
    onToggleMod,
    onRejoinTeam
  }: Props = $props();

  let visibleBuckets = $derived(visiblePlayerBuckets(players));
</script>

{#snippet SpyIcon()}
  <SvgMaskIcon src={customSvg.spy} classes="h-3.5 w-3.5" />
{/snippet}

{#snippet RepIcon()}
  <SvgMaskIcon src={customSvg.representative} classes="h-3.5 w-3.5" />
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
  <article class="group rounded-2xl border border-slate-700 bg-slate-950/85 p-3 transition duration-300 hover:-translate-y-0.5 hover:border-slate-500">
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
    <div class="mt-3 flex flex-wrap gap-2">
      {#if phase === 'lobby' && (hostControls || player.id === (viewer?.playerId || viewer?.userId))}
        <button class={['rounded-full border px-3 py-1.5 text-xs font-bold transition', player.team === 'blue' ? 'text-white' : 'text-slate-100/70']} style={`border-color: ${hexWithAlpha(teamColor('blue', settings), player.team === 'blue' ? 'cc' : '80')}; background-color: ${player.team === 'blue' ? hexWithAlpha(teamColor('blue', settings), '40') : 'transparent'};`} onclick={() => onAssignTeam(player.id, 'blue')}>{displayTeamName('blue', settings)}</button>
        <button class={['rounded-full border px-3 py-1.5 text-xs font-bold transition', player.team === 'red' ? 'text-white' : 'text-slate-100/70']} style={`border-color: ${hexWithAlpha(teamColor('red', settings), player.team === 'red' ? 'cc' : '80')}; background-color: ${player.team === 'red' ? hexWithAlpha(teamColor('red', settings), '40') : 'transparent'};`} onclick={() => onAssignTeam(player.id, 'red')}>{displayTeamName('red', settings)}</button>
        <button class={['rounded-full border px-3 py-1.5 text-xs font-bold transition', player.team === 'observers' ? 'border-slate-300 bg-slate-400/20 text-slate-100' : 'border-slate-300/50 text-slate-100/60 hover:bg-slate-400/20']} onclick={() => onAssignTeam(player.id, 'observers')}>Observer</button>
      {/if}
      {#if phase === 'lobby' && player.team === 'observers' && player.previousTeam && (hostControls || player.id === (viewer?.playerId || viewer?.userId))}
        <button class="rounded-full border border-emerald-300/70 px-3 py-1.5 text-xs font-black text-emerald-100 hover:bg-emerald-300/10" onclick={() => onRejoinTeam(player.id)}>
          Rejoin {displayTeamName(player.previousTeam, settings)}
        </button>
      {/if}
      {#if phase === 'lobby' && hostControls && (player.team === 'blue' || player.team === 'red')}
        <button class={['rounded-full border px-3 py-1.5 text-xs font-bold transition', player.spymaster ? 'border-slate-100 bg-white text-slate-950' : 'border-slate-600 text-slate-200 hover:border-slate-300']} onclick={() => onToggleSpymaster(player.id)}>Spy</button>
        <button class={['rounded-full border px-3 py-1.5 text-xs font-bold transition', player.representative ? 'border-amber-200 bg-amber-200 text-slate-950' : 'border-slate-600 text-slate-200 hover:border-slate-300']} onclick={() => onToggleRepresentative(player.id)}>Rep</button>
      {/if}
      {#if phase === 'lobby' && hostControls && player.id !== roomHostId}
        <button class="rounded-full border border-emerald-400/60 px-3 py-1.5 text-xs font-bold text-emerald-100 hover:bg-emerald-400/15" onclick={() => onToggleMod(player.id)}>
          {player.mod ? 'Demote mod' : 'Promote mod'}
        </button>
      {/if}
    </div>
  </article>
{/snippet}

{#snippet TeamColumn(tone: 'blue' | 'red' | 'observers' | 'unassigned', members: LobbyPlayer[])}
  {@const title = tone === 'blue' ? displayTeamName('blue', settings) : tone === 'red' ? displayTeamName('red', settings) : tone === 'observers' ? 'Observers' : 'Unassigned'}
  <section class={['rounded-[1.5rem] border p-3 shadow-2xl shadow-slate-950/25 sm:p-4',
    tone === 'blue' ? 'border-blue-300/30 bg-blue-400/10' : 
    tone === 'red' ? 'border-red-300/30 bg-red-400/10' : 
    tone === 'observers' ? 'border-slate-500/30 bg-slate-700/10' :
    'border-slate-700 bg-slate-900/40']}
    style={tone === 'blue' || tone === 'red' ? `border-color: ${hexWithAlpha(teamColor(tone, settings), '55')}; background-color: ${hexWithAlpha(teamColor(tone, settings), '18')};` : ''}>
    {#if tone === 'observers' || tone === 'unassigned'}
      <h2 class="text-lg font-black tracking-tight">{title} ({members.length})</h2>
    {:else}
      <h2 class="sr-only">{title}</h2>
      <p class="text-xs font-black uppercase tracking-[0.18em] text-slate-300">{members.length} players</p>
    {/if}
    <div class="mt-3 grid gap-2">
      {#each members as player (player.id)}
        {@render PlayerCard(player)}
      {:else}
        <p class="rounded-2xl border border-slate-700/70 bg-slate-950/70 px-4 py-6 text-center text-sm text-slate-400">Empty.</p>
      {/each}
    </div>
  </section>
{/snippet}

<div id="players" class="space-y-4">
  <div class="grid grid-flow-dense gap-4 md:grid-cols-2">
    {#each visibleBuckets as bucket (bucket.tone)}
      {@render TeamColumn(bucket.tone, bucket.members)}
    {/each}
  </div>
</div>
