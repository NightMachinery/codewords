<script lang="ts">
  import { onDestroy, onMount } from 'svelte';

  import { api, defaultSettings, type Settings, type Viewer, type Wordpack } from '../lib/api';
  import { copyText } from '../lib/clipboard';
  import { getOrCreateAuthToken, resolveSessionCredential, type SessionCredential } from '../lib/identity';
  import { canManageLobby, playerBuckets, startReadiness, type LobbyPlayer } from '../lib/lobby';
  import { RoomSocket, type RoomSocketMessage } from '../lib/realtime';
  import { roomIdFromPath, roomPath, websocketRoomUrl } from '../lib/routes';

  let roomId = $state('');
  let authToken = '';
  let credential: SessionCredential | null = null;
  let credentialMode = $state<'none' | 'auth' | 'migrate'>('none');
  let displayName = $state('');
  let nameDraft = $state('');
  let players = $state<LobbyPlayer[]>([]);
  let viewer = $state<Viewer | null>(null);
  let settings = $state<Settings>({ ...defaultSettings });
  let wordpacks = $state<Wordpack[]>([]);
  let phase = $state('lobby');
  let loading = $state(true);
  let savingName = $state(false);
  let connection = $state('disconnected');
  let error = $state('');
  let copyStatus = $state('');
  let migrateUrl = $state('');
  let socket: RoomSocket | null = null;

  let buckets = $derived(playerBuckets(players));
  let startState = $derived(startReadiness(players));
  let hostControls = $derived(canManageLobby(viewer));
  let currentPlayer = $derived(players.find((player) => player.id === viewer?.userId || player.id === viewer?.playerId));
  let needsName = $derived(Boolean(credentialMode === 'auth' && !displayName));

  onMount(() => {
    void boot();
    return () => socket?.close();
  });

  onDestroy(() => socket?.close());

  async function boot() {
    try {
      roomId = roomIdFromPath(window.location.pathname);
      authToken = getOrCreateAuthToken(localStorage);
      credential = resolveSessionCredential(new URL(window.location.href), localStorage);
      credentialMode = credential.mode;
      const packs = await api.wordpacks();
      wordpacks = packs.wordpacks;

      if (credential.mode === 'migrate') {
        const migrated = await api.migrateBootstrap(roomId, credential.migrateId);
        displayName = migrated.displayName;
        nameDraft = migrated.displayName;
      } else {
        const identity = await api.bootstrap(authToken);
        displayName = identity.displayName;
        nameDraft = identity.displayName;
      }

      const room = await api.getRoom(roomId, credential);
      viewer = room.viewer;
      settings = { ...defaultSettings, ...room.settings };
      if (credential.mode === 'auth' && displayName) {
        await api.joinRoom(roomId, authToken, displayName);
      }
      if (!needsName) {
        connectSocket();
      }
    } catch (err) {
      error = err instanceof Error ? err.message : 'Could not load this room.';
    } finally {
      loading = false;
    }
  }

  async function saveNameAndJoin() {
    const name = nameDraft.trim();
    if (!name) {
      error = 'Choose a display name to join this room.';
      return;
    }
    savingName = true;
    error = '';
    try {
      const saved = await api.saveDisplayName(authToken, name);
      displayName = saved.displayName;
      nameDraft = saved.displayName;
      const joined = await api.joinRoom(roomId, authToken, saved.displayName);
      viewer = joined.viewer;
      connectSocket();
    } catch (err) {
      error = err instanceof Error ? err.message : 'Could not join this room.';
    } finally {
      savingName = false;
    }
  }

  function connectSocket() {
    socket?.close();
    if (!credential) return;
    socket = new RoomSocket(websocketRoomUrl(new URL(window.location.href), roomId, credential), {
      onStatus: (status) => {
        connection = status;
      },
      onMessage: handleSocketMessage,
    });
    socket.connect();
  }

  function handleSocketMessage(message: RoomSocketMessage) {
    if (message.type === 'snapshot') {
      players = message.snapshot.players;
      settings = { ...defaultSettings, ...message.snapshot.settings };
      viewer = message.snapshot.viewer;
      phase = message.snapshot.phase;
    }
    if (message.type === 'error') {
      error = message.message;
    }
  }

  function assignTeam(playerId: string, team: 'blue' | 'red') {
    socket?.send({ type: 'assignTeam', playerId, team });
  }

  function toggleSpymaster(playerId: string) {
    socket?.send({ type: 'toggleSpymaster', playerId });
  }

  function toggleRepresentative(playerId: string) {
    socket?.send({ type: 'toggleRepresentative', playerId });
  }

  async function saveSettings() {
    if (!authToken) return;
    error = '';
    try {
      await api.updateSettings(roomId, authToken, settings);
    } catch (err) {
      error = err instanceof Error ? err.message : 'Could not save settings.';
    }
  }

  function startGame() {
    error = '';
    socket?.send({ type: 'startGame' });
  }

  async function copyRoomLink() {
    const link = `${window.location.origin}${roomPath(roomId)}`;
    const result = await copyText(link);
    copyStatus = result.ok ? 'Room link copied.' : link;
  }

  async function copyMigrateLink() {
    error = '';
    try {
      const link = await api.createMigrateLink(roomId, authToken);
      migrateUrl = link.migrateUrl;
      const result = await copyText(link.migrateUrl);
      copyStatus = result.ok ? 'Migrate-device link copied.' : link.migrateUrl;
    } catch (err) {
      error = err instanceof Error ? err.message : 'Could not create a migrate link.';
    }
  }
</script>

<main class="min-h-screen w-full overflow-x-hidden bg-[oklch(14%_0.018_255)] text-slate-100">
  <div class="mx-auto w-full max-w-7xl px-5 py-6 sm:px-8">
    <nav class="flex flex-wrap items-center justify-between gap-3 rounded-full border border-slate-700/70 bg-slate-900/75 px-5 py-3 shadow-2xl shadow-slate-950/40">
      <a class="text-lg font-black tracking-tight text-slate-50" href="/">Codewords</a>
      <div class="flex items-center gap-3 text-sm text-slate-300">
        <span class={['h-2.5 w-2.5 rounded-full', connection === 'connected' ? 'bg-emerald-300' : 'bg-amber-300']}></span>
        <span>{connection}</span>
      </div>
    </nav>

    {#if loading}
      <section class="grid min-h-[70vh] place-items-center">
        <p class="text-slate-300">Loading room...</p>
      </section>
    {:else if needsName}
      <section class="mx-auto grid min-h-[70vh] max-w-md place-items-center">
        <div class="w-full rounded-[2rem] border border-slate-700 bg-slate-900 p-6 shadow-2xl shadow-slate-950/40">
          <h1 class="text-3xl font-black tracking-tight">Join this room</h1>
          <p class="mt-3 text-slate-300">Pick the display name other players will see.</p>
          <input
            class="mt-6 w-full rounded-2xl border border-slate-700 bg-slate-950 px-4 py-3 text-slate-50 outline-none ring-emerald-300 transition focus:ring-2"
            bind:value={nameDraft}
            maxlength="40"
            placeholder="Your table name"
          />
          <button
            class="mt-4 w-full rounded-2xl bg-emerald-300 px-5 py-3 font-black text-slate-950 transition hover:bg-emerald-200 disabled:opacity-60"
            disabled={savingName}
            onclick={saveNameAndJoin}
          >
            {savingName ? 'Joining...' : 'Join lobby'}
          </button>
        </div>
      </section>
    {:else}
      <header class="grid gap-8 py-12 lg:grid-cols-[1fr_22rem] lg:py-16">
        <div class="max-w-5xl">
          <p class="mb-4 text-sm font-semibold uppercase tracking-[0.22em] text-emerald-300">Room {roomId}</p>
          <h1 class="max-w-5xl text-5xl font-black leading-[0.96] tracking-[-0.05em] text-slate-50 sm:text-7xl">
            Gather teams, choose roles, then start.
          </h1>
          <p class="mt-6 max-w-2xl text-lg leading-8 text-slate-300">
            {displayName ? `You are ${displayName}. ` : ''}Share the room link with players on this server. Use migrate-device for the same seat on another browser.
          </p>
        </div>

        <aside class="self-start rounded-[2rem] border border-slate-700/70 bg-slate-900/80 p-5 shadow-2xl shadow-slate-950/40">
          <div class="grid gap-3">
            <button class="rounded-2xl bg-slate-100 px-5 py-3 font-black text-slate-950 transition hover:bg-white" onclick={copyRoomLink}>Copy room link</button>
            {#if currentPlayer}
              <button class="rounded-2xl border border-slate-600 px-5 py-3 font-bold text-slate-100 transition hover:border-emerald-300 hover:text-emerald-200" onclick={copyMigrateLink}>
                Copy migrate-device link
              </button>
            {/if}
            {#if copyStatus}
              <p class="break-all rounded-2xl border border-emerald-300/40 bg-emerald-300/10 px-4 py-3 text-sm text-emerald-100">{copyStatus}</p>
            {/if}
            {#if migrateUrl && copyStatus === migrateUrl}
              <p class="break-all text-xs text-slate-400">{migrateUrl}</p>
            {/if}
          </div>
        </aside>
      </header>

      {#if phase !== 'lobby'}
        <section class="rounded-[2rem] border border-emerald-300/30 bg-emerald-300/10 p-6 text-emerald-50">
          The match has started. Gameplay UI arrives in Milestone 6.
        </section>
      {/if}

      <section class="grid gap-6 lg:grid-cols-[1fr_22rem]">
        <div class="space-y-6">
          <div class="grid grid-flow-dense gap-6 md:grid-cols-2">
            {@render TeamColumn('Blue team', 'blue', buckets.blue, viewer, hostControls, assignTeam, toggleSpymaster, toggleRepresentative)}
            {@render TeamColumn('Red team', 'red', buckets.red, viewer, hostControls, assignTeam, toggleSpymaster, toggleRepresentative)}
          </div>

          <section class="rounded-[2rem] border border-slate-700/70 bg-slate-900/70 p-5">
            <h2 class="text-xl font-black tracking-tight">Unassigned</h2>
            <div class="mt-4 grid gap-3 sm:grid-cols-2">
              {#each buckets.unassigned as player (player.id)}
                {@render PlayerCard(player, viewer, hostControls, assignTeam, toggleSpymaster, toggleRepresentative)}
              {:else}
                <p class="text-sm text-slate-400">No unassigned players.</p>
              {/each}
            </div>
          </section>
        </div>

        <aside class="space-y-6">
          <section class="rounded-[2rem] border border-slate-700/70 bg-slate-900/80 p-5 shadow-2xl shadow-slate-950/30">
            <h2 class="text-xl font-black tracking-tight">Host settings</h2>
            <fieldset class="mt-5 space-y-4 disabled:opacity-60" disabled={!hostControls}>
              <label class="block">
                <span class="text-sm font-semibold text-slate-200">Wordpack</span>
                <select class="mt-2 w-full rounded-2xl border border-slate-700 bg-slate-950 px-4 py-3 text-slate-50" bind:value={settings.wordpackId} onchange={saveSettings}>
                  {#each wordpacks as pack (pack.id)}
                    <option value={pack.id}>{pack.label} ({pack.wordCount})</option>
                  {/each}
                </select>
              </label>
              <label class="block">
                <span class="text-sm font-semibold text-slate-200">Black cards</span>
                <input class="mt-2 w-full rounded-2xl border border-slate-700 bg-slate-950 px-4 py-3 text-slate-50" type="number" min="0" max="8" bind:value={settings.blackCards} onchange={saveSettings} />
              </label>
              <label class="flex items-center gap-3 rounded-2xl border border-slate-700 bg-slate-950 px-4 py-3">
                <input type="checkbox" bind:checked={settings.enforceClueGuessLimit} onchange={saveSettings} />
                <span class="text-sm text-slate-200">Require clue before guessing</span>
              </label>
              <label class="flex items-center gap-3 rounded-2xl border border-slate-700 bg-slate-950 px-4 py-3">
                <input type="checkbox" bind:checked={settings.allowInfinityClue} onchange={saveSettings} />
                <span class="text-sm text-slate-200">Allow infinity clues</span>
              </label>
            </fieldset>
          </section>

          <section class="rounded-[2rem] border border-slate-700/70 bg-slate-900/80 p-5">
            <h2 class="text-xl font-black tracking-tight">Start match</h2>
            <p class="mt-3 text-sm leading-6 text-slate-300">{startState.ready ? 'The lobby is ready.' : startState.reason}</p>
            <button
              class="mt-5 w-full rounded-2xl bg-emerald-300 px-5 py-3 font-black text-slate-950 transition hover:bg-emerald-200 disabled:cursor-not-allowed disabled:opacity-50"
              disabled={!hostControls || !startState.ready}
              onclick={startGame}
            >
              Start game
            </button>
          </section>
        </aside>
      </section>
    {/if}

    {#if error}
      <p class="mt-6 rounded-2xl border border-red-400/40 bg-red-400/10 px-4 py-3 text-sm text-red-100">{error}</p>
    {/if}
  </div>
</main>

{#snippet roleBadges(player: LobbyPlayer)}
  {#if player.spymaster}
    <span class="rounded-full bg-slate-100 px-2.5 py-1 text-xs font-black text-slate-950">Spymaster</span>
  {/if}
  {#if player.representative}
    <span class="rounded-full bg-amber-200 px-2.5 py-1 text-xs font-black text-slate-950">Representative</span>
  {/if}
{/snippet}

{#snippet PlayerCard(player: LobbyPlayer, viewer: Viewer | null, hostControls: boolean, onassign: (id: string, team: 'blue' | 'red') => void, ontoggleSpy: (id: string) => void, ontoggleRep: (id: string) => void)}
  <article class="group rounded-2xl border border-slate-700 bg-slate-950 p-4 transition duration-300 hover:-translate-y-0.5 hover:border-slate-500">
    <div class="flex items-start justify-between gap-3">
      <div>
        <h3 class="font-black text-slate-50">{player.displayName || 'Unnamed player'}</h3>
        <p class="text-xs text-slate-500">{player.id === viewer?.userId ? 'You' : player.id.slice(0, 8)}</p>
      </div>
      <div class="flex flex-wrap justify-end gap-2">{@render roleBadges(player)}</div>
    </div>
    <div class="mt-4 flex flex-wrap gap-2">
      {#if hostControls || player.id === viewer?.userId}
        <button class="rounded-full border border-blue-300/50 px-3 py-1.5 text-xs font-bold text-blue-100 hover:bg-blue-400/20" onclick={() => onassign(player.id, 'blue')}>Blue</button>
        <button class="rounded-full border border-red-300/50 px-3 py-1.5 text-xs font-bold text-red-100 hover:bg-red-400/20" onclick={() => onassign(player.id, 'red')}>Red</button>
      {/if}
      {#if hostControls && player.team}
        <button class="rounded-full border border-slate-600 px-3 py-1.5 text-xs font-bold text-slate-200 hover:border-slate-300" onclick={() => ontoggleSpy(player.id)}>Spy</button>
        <button class="rounded-full border border-slate-600 px-3 py-1.5 text-xs font-bold text-slate-200 hover:border-slate-300" onclick={() => ontoggleRep(player.id)}>Rep</button>
      {/if}
    </div>
  </article>
{/snippet}

{#snippet TeamColumn(title: string, tone: 'blue' | 'red', players: LobbyPlayer[], viewer: Viewer | null, hostControls: boolean, onassign: (id: string, team: 'blue' | 'red') => void, ontoggleSpy: (id: string) => void, ontoggleRep: (id: string) => void)}
  <section class={['rounded-[2rem] border p-5 shadow-2xl shadow-slate-950/25', tone === 'blue' ? 'border-blue-300/30 bg-blue-400/10' : 'border-red-300/30 bg-red-400/10']}>
    <h2 class="text-xl font-black tracking-tight">{title}</h2>
    <div class="mt-4 grid gap-3">
      {#each players as player (player.id)}
        {@render PlayerCard(player, viewer, hostControls, onassign, ontoggleSpy, ontoggleRep)}
      {:else}
        <p class="rounded-2xl border border-slate-700/70 bg-slate-950/70 px-4 py-6 text-center text-sm text-slate-400">Waiting for players.</p>
      {/each}
    </div>
  </section>
{/snippet}
