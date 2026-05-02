<script lang="ts">
  import { onDestroy, onMount } from 'svelte';

  import { api, defaultSettings, type ChatMessage, type PictureAsset, type Settings, type Viewer, type Wordpack } from '../lib/api';
  import { copyText } from '../lib/clipboard';
  import {
    canSubmitClue,
    cardContentLabel,
    cardImageUrl,
    cardModeFromImageCount,
    cardViewState,
    clueSubmitProblem,
    defaultGameplayPreferences,
    findViewerPlayer,
    formatClueNumber,
    imageCountForMode,
    parseClueNumber,
    readGameplayPreferences,
    shouldAutoJoinRoom,
    viewerRole,
    writeGameplayPreferences,
    type ClueEntry,
    type GameplayCard,
    type GameplayPreferences,
    type LastSelected,
    type RemainingCounts,
  } from '../lib/gameplay';
  import { getOrCreateAuthToken, resolveSessionCredential, type SessionCredential } from '../lib/identity';
  import { canManageLobby, playerBuckets, startReadiness, type LobbyPlayer } from '../lib/lobby';
  import { RoomSocket, type RoomSocketMessage } from '../lib/realtime';
  import { roomIdFromPath, roomPath, websocketRoomUrl } from '../lib/routes';

  let roomId = $state('');
  let roomStatus = $state('lobby');
  let roomHostId = $state('');
  let authToken = '';
  let credential: SessionCredential | null = null;
  let credentialMode = $state<'none' | 'auth' | 'migrate'>('none');
  let displayName = $state('');
  let nameDraft = $state('');
  let players = $state<LobbyPlayer[]>([]);
  let viewer = $state<Viewer | null>(null);
  let settings = $state<Settings>({ ...defaultSettings });
  let wordpacks = $state<Wordpack[]>([]);
  let pictures = $state<PictureAsset[]>([]);
  let pictureCatalogAvailable = $state(false);
  let phase = $state<'lobby' | 'active' | 'game_over'>('lobby');
  let currentTeam = $state<'blue' | 'red' | ''>('');
  let winner = $state<'blue' | 'red' | ''>('');
  let cards = $state<GameplayCard[]>([]);
  let lastSelected = $state<LastSelected | null>(null);
  let remainingCounts = $state<RemainingCounts>({ blue: 0, red: 0 });
  let clueLog = $state<ClueEntry[]>([]);
  let clueText = $state('');
  let clueNumber = $state('');
  let chatMessages = $state<ChatMessage[]>([]);
  let chatDraft = $state('');
  let preferences = $state<GameplayPreferences>({ ...defaultGameplayPreferences });
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
  let currentPlayer = $derived(findViewerPlayer(players, viewer));
  let needsName = $derived(Boolean(credentialMode === 'auth' && !displayName && roomStatus === 'lobby'));
  let role = $derived(viewerRole(players, viewer, currentTeam, phase));
  let cluePermission = $derived(canSubmitClue(players, viewer, currentTeam, phase));
  let clueNumberParsed = $derived(parseClueNumber(clueNumber));
  let clueProblem = $derived(clueSubmitProblem(clueText, clueNumberParsed, settings));
  let currentClue = $derived(clueLog.slice().reverse().find((entry) => entry.status === 'active') ?? null);
  let guessProblem = $derived(guessDisabledReason());
  let passProblem = $derived(passDisabledReason());
  let cardMode = $derived(cardModeFromImageCount(settings.imageCardCount ?? 0));

  onMount(() => {
    void boot();
    return () => socket?.close();
  });

  onDestroy(() => socket?.close());

  async function boot() {
    try {
      roomId = roomIdFromPath(window.location.pathname);
      authToken = getOrCreateAuthToken(localStorage);
      preferences = readGameplayPreferences(localStorage);
      credential = resolveSessionCredential(new URL(window.location.href), localStorage);
      credentialMode = credential.mode;
      const packs = await api.wordpacks();
      wordpacks = packs.wordpacks;
      const pictureCatalog = await api.pictureCatalog();
      pictureCatalogAvailable = pictureCatalog.available;
      pictures = pictureCatalog.images;

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
      roomStatus = room.room.status;
      roomHostId = room.room.hostUserId;
      viewer = room.viewer;
      settings = { ...defaultSettings, ...room.settings };
      chatMessages = room.chatMessages ?? [];
      if (shouldAutoJoinRoom(room.room, credential.mode, displayName)) {
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
      if (roomStatus === 'lobby') {
        const joined = await api.joinRoom(roomId, authToken, saved.displayName);
        viewer = joined.viewer;
      } else {
        viewer = { userId: saved.userId, isHost: saved.userId === roomHostId };
      }
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
      currentTeam = message.snapshot.currentTeam;
      winner = message.snapshot.winner;
      cards = message.snapshot.cards ?? [];
      lastSelected = message.snapshot.lastSelected ?? null;
      remainingCounts = message.snapshot.remainingCounts ?? { blue: 0, red: 0 };
      clueLog = message.snapshot.clueLog ?? [];
    }
    if (message.type === 'chatMessage') {
      chatMessages = [...chatMessages, message.message].slice(-50);
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

  function setCardMode(mode: 'words' | 'images' | 'mixed') {
    settings = { ...settings, imageCardCount: imageCountForMode(mode, settings.imageCardCount) };
    void saveSettings();
  }

  function setMixedImageCount(value: string) {
    const parsed = Number.parseInt(value, 10);
    settings = { ...settings, imageCardCount: Math.min(24, Math.max(1, Number.isFinite(parsed) ? parsed : 8)) };
    void saveSettings();
  }

  function submitClue() {
    error = '';
    if (!cluePermission.allowed) {
      error = cluePermission.reason;
      return;
    }
    if (clueProblem) {
      error = clueProblem;
      return;
    }
    socket?.send({ type: 'submitClue', text: clueText, number: clueNumber });
  }

  function guessCard(index: number, card: GameplayCard) {
    error = '';
    const reason = guessDisabledReason(card);
    if (reason) {
      error = reason;
      return;
    }
    if (preferences.confirmGuesses && !window.confirm(`Reveal ${cardContentLabel(card)}?`)) {
      return;
    }
    socket?.send({ type: 'guessCard', index });
  }

  function passTurn() {
    error = '';
    const reason = passDisabledReason();
    if (reason) {
      error = reason;
      return;
    }
    if (preferences.confirmPasses && !window.confirm('Pass this turn?')) {
      return;
    }
    socket?.send({ type: 'passTurn' });
  }

  function guessDisabledReason(card?: GameplayCard): string {
    if (phase === 'game_over') return 'The match is over.';
    if (phase !== 'active') return 'The match has not started.';
    if (!role.player) return 'Spectators are read-only.';
    if (!role.activeGuesser) {
      if (role.kind === 'spymaster') return 'Spymasters cannot guess while teammates can.';
      return `Only the ${currentTeam} guesser can reveal cards.`;
    }
    if (settings.enforceClueGuessLimit && (!currentClue || currentClue.number.kind === 'blank')) return 'Wait for a numbered clue first.';
    if (card?.revealed) return 'That card is already revealed.';
    return '';
  }

  function passDisabledReason(): string {
    if (phase === 'game_over') return 'The match is over.';
    if (phase !== 'active') return 'The match has not started.';
    if (!role.player) return 'Spectators are read-only.';
    if (!role.activeGuesser) return role.kind === 'spymaster' ? 'Spymasters cannot pass while teammates can.' : `Only the ${currentTeam} guesser can pass.`;
    return '';
  }

  function updatePreferences(next: Partial<GameplayPreferences>) {
    preferences = { ...preferences, ...next };
    writeGameplayPreferences(localStorage, preferences);
  }

  function sendChat() {
    const body = chatDraft.trim();
    if (!body) return;
    if (!currentPlayer) {
      error = 'Anonymous spectators can read chat but cannot send messages.';
      return;
    }
    socket?.send({ type: 'sendChat', body });
    chatDraft = '';
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
            {phase === 'lobby' ? 'Gather teams, choose roles, then start.' : phase === 'game_over' ? `${winner || 'A team'} wins the board.` : `${currentTeam || 'Current'} team is on the clock.`}
          </h1>
          <p class="mt-6 max-w-2xl text-lg leading-8 text-slate-300">
            {displayName ? `You are ${displayName}. ` : ''}{phase === 'lobby' ? 'Share the room link with players on this server. Use migrate-device for the same seat on another browser.' : role.kind === 'spectator' ? 'You are watching as a read-only spectator.' : role.kind === 'spymaster' ? 'You can see hidden colors and submit clues on your team turn.' : 'You see unrevealed cards until a spymaster or guess reveals them.'}
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

      {#if phase === 'lobby'}
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
              <section class="rounded-2xl border border-slate-700 bg-slate-950 p-4">
                <span class="text-sm font-semibold text-slate-200">Card content</span>
                <div class="mt-3 grid gap-2">
                  <label class="flex items-center gap-3 text-sm text-slate-200">
                    <input type="radio" name="card-mode" checked={cardMode === 'words'} onchange={() => setCardMode('words')} />
                    Words only
                  </label>
                  <label class="flex items-center gap-3 text-sm text-slate-200">
                    <input type="radio" name="card-mode" checked={cardMode === 'images'} disabled={!pictureCatalogAvailable} onchange={() => setCardMode('images')} />
                    Images only {pictureCatalogAvailable ? `(${pictures.length} available)` : '(no local pictures)'}
                  </label>
                  <label class="flex items-center gap-3 text-sm text-slate-200">
                    <input type="radio" name="card-mode" checked={cardMode === 'mixed'} disabled={!pictureCatalogAvailable} onchange={() => setCardMode('mixed')} />
                    Mixed images and words
                  </label>
                </div>
                {#if cardMode === 'mixed'}
                  <label class="mt-3 block">
                    <span class="text-xs font-semibold text-slate-400">Image cards: {settings.imageCardCount}</span>
                    <input class="mt-2 w-full" type="range" min="1" max="24" value={settings.imageCardCount} oninput={(event) => setMixedImageCount(event.currentTarget.value)} />
                  </label>
                {/if}
                {#if settings.imageCardCount > pictures.length}
                  <p class="mt-3 text-xs text-amber-100">This server only has {pictures.length} local pictures. Add files to the configured pictures directory or choose fewer image cards.</p>
                {/if}
              </section>

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

          {@render ChatPanel(chatMessages, chatDraft, currentPlayer, sendChat)}

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
      {:else}
        <section class="grid gap-6 xl:grid-cols-[1fr_24rem]">
          <div class="space-y-6">
            {#if phase === 'game_over'}
              <section class={['rounded-[2rem] border p-6 shadow-2xl shadow-slate-950/30', winner === 'blue' ? 'border-blue-300/50 bg-blue-400/15' : 'border-red-300/50 bg-red-400/15']}>
                <p class="text-sm font-black uppercase tracking-[0.25em] text-emerald-200">Game over</p>
                <h2 class="mt-2 text-4xl font-black tracking-[-0.04em] text-slate-50">{winner === 'blue' ? 'Blue' : 'Red'} team wins</h2>
                <p class="mt-3 text-slate-300">All card colors are now revealed to every viewer.</p>
              </section>
            {/if}

            <section class="rounded-[2rem] border border-slate-700/70 bg-slate-900/70 p-4 shadow-2xl shadow-slate-950/35 sm:p-5">
              <div class="mb-5 flex flex-wrap items-center justify-between gap-3">
                <div>
                  <p class="text-xs font-black uppercase tracking-[0.22em] text-slate-400">Board</p>
                  <h2 class="text-2xl font-black tracking-tight">5 × 5 code grid</h2>
                </div>
                <div class="flex flex-wrap gap-2 text-sm font-black">
                  <span class="rounded-full border border-blue-300/40 bg-blue-400/15 px-3 py-1.5 text-blue-100">Blue left {remainingCounts.blue}</span>
                  <span class="rounded-full border border-red-300/40 bg-red-400/15 px-3 py-1.5 text-red-100">Red left {remainingCounts.red}</span>
                </div>
              </div>

              <div class="grid grid-cols-2 gap-2 sm:grid-cols-3 md:grid-cols-5 md:gap-3">
                {#each cards as card, index (`${card.word ?? card.imageId ?? 'card'}-${index}`)}
                  {@const view = cardViewState(card, index, role.canSeeHiddenColors, lastSelected)}
                  <button
                    class={['group min-h-28 rounded-[1.35rem] border p-3 text-left shadow-xl shadow-slate-950/25 transition duration-200 hover:-translate-y-0.5 disabled:cursor-not-allowed disabled:hover:translate-y-0', view.classes, !role.activeGuesser || card.revealed || phase !== 'active' ? 'disabled:opacity-80' : ''].join(' ')}
                    disabled={Boolean(guessDisabledReason(card))}
                    title={guessDisabledReason(card) || `Reveal ${cardContentLabel(card)}`}
                    onclick={() => guessCard(index, card)}
                  >
                    <span class="block text-[0.65rem] font-black uppercase tracking-[0.18em] opacity-70">{view.label}</span>
                    {#if card.contentType === 'image'}
                      <img class="mt-3 aspect-[2/3] w-full rounded-2xl object-cover" src={cardImageUrl(card)} alt="Card illustration" loading="lazy" />
                    {:else}
                      <span class="mt-4 block break-words text-xl font-black uppercase tracking-[0.04em] sm:text-2xl">{card.word ?? 'Card'}</span>
                    {/if}
                    {#if view.isLastSelected}
                      <span class="mt-3 inline-flex rounded-full bg-emerald-200 px-2.5 py-1 text-xs font-black text-slate-950">Last pick</span>
                    {/if}
                  </button>
                {:else}
                  <p class="col-span-full rounded-2xl border border-slate-700 bg-slate-950 p-6 text-slate-300">Waiting for the board snapshot...</p>
                {/each}
              </div>
            </section>
          </div>

          <aside class="space-y-6">
            <section class={['rounded-[2rem] border p-5 shadow-2xl shadow-slate-950/30', currentTeam === 'blue' ? 'border-blue-300/40 bg-blue-400/10' : 'border-red-300/40 bg-red-400/10']}>
              <p class="text-xs font-black uppercase tracking-[0.22em] text-slate-400">Turn</p>
              <h2 class="mt-1 text-3xl font-black tracking-tight capitalize">{currentTeam} team</h2>
              {#if currentClue}
                <p class="mt-4 rounded-2xl border border-slate-600 bg-slate-950/70 px-4 py-3">
                  <span class="block text-xs font-black uppercase tracking-[0.18em] text-slate-400">Current clue</span>
                  <span class="text-lg font-black">{currentClue.text} · {formatClueNumber(currentClue.number)}</span>
                  <span class="block text-xs text-slate-400">{currentClue.guesses} guesses taken</span>
                </p>
              {:else if settings.enforceClueGuessLimit}
                <p class="mt-4 rounded-2xl border border-amber-200/40 bg-amber-200/10 px-4 py-3 text-sm text-amber-100">A numbered clue is required before guessing.</p>
              {/if}
            </section>

            <section class="rounded-[2rem] border border-slate-700/70 bg-slate-900/80 p-5">
              <h2 class="text-xl font-black tracking-tight">Clue composer</h2>
              <fieldset class="mt-4 space-y-3 disabled:opacity-55" disabled={!cluePermission.allowed || phase !== 'active'}>
                <input
                  class="w-full rounded-2xl border border-slate-700 bg-slate-950 px-4 py-3 font-semibold text-slate-50 outline-none ring-emerald-300 transition focus:ring-2"
                  bind:value={clueText}
                  maxlength="40"
                  placeholder="One-word clue"
                />
                <select class="w-full rounded-2xl border border-slate-700 bg-slate-950 px-4 py-3 text-slate-50" bind:value={clueNumber}>
                  <option value="">Any / blank</option>
                  {#each [1, 2, 3, 4, 5, 6, 7, 8, 9] as n (n)}
                    <option value={String(n)}>{n}</option>
                  {/each}
                  {#if settings.allowInfinityClue}
                    <option value="∞">∞</option>
                  {/if}
                </select>
                <button class="w-full rounded-2xl bg-emerald-300 px-5 py-3 font-black text-slate-950 transition hover:bg-emerald-200 disabled:cursor-not-allowed disabled:opacity-50" disabled={Boolean(clueProblem)} onclick={submitClue}>
                  Submit / update clue
                </button>
              </fieldset>
              <p class="mt-3 text-sm leading-6 text-slate-400">{cluePermission.allowed ? clueProblem || 'Your latest clue replaces the active clue for this turn.' : cluePermission.reason}</p>
            </section>

            <section class="rounded-[2rem] border border-slate-700/70 bg-slate-900/80 p-5">
              <h2 class="text-xl font-black tracking-tight">Guess controls</h2>
              <p class="mt-2 text-sm leading-6 text-slate-300">{guessProblem || 'Select an unrevealed board card to guess.'}</p>
              <button class="mt-4 w-full rounded-2xl border border-slate-500 px-5 py-3 font-black text-slate-100 transition hover:border-emerald-300 hover:text-emerald-200 disabled:cursor-not-allowed disabled:opacity-50" disabled={Boolean(passProblem)} onclick={passTurn}>
                Pass turn
              </button>
              {#if passProblem}
                <p class="mt-2 text-xs text-slate-500">{passProblem}</p>
              {/if}
            </section>

            <section class="rounded-[2rem] border border-slate-700/70 bg-slate-900/80 p-5">
              <h2 class="text-xl font-black tracking-tight">Local confirmations</h2>
              <label class="mt-4 flex items-center gap-3 rounded-2xl border border-slate-700 bg-slate-950 px-4 py-3">
                <input type="checkbox" checked={preferences.confirmGuesses} onchange={(event) => updatePreferences({ confirmGuesses: event.currentTarget.checked })} />
                <span class="text-sm text-slate-200">Confirm before revealing a card</span>
              </label>
              <label class="mt-3 flex items-center gap-3 rounded-2xl border border-slate-700 bg-slate-950 px-4 py-3">
                <input type="checkbox" checked={preferences.confirmPasses} onchange={(event) => updatePreferences({ confirmPasses: event.currentTarget.checked })} />
                <span class="text-sm text-slate-200">Confirm before passing</span>
              </label>
            </section>

            {@render ChatPanel(chatMessages, chatDraft, currentPlayer, sendChat)}

            <section class="rounded-[2rem] border border-slate-700/70 bg-slate-900/80 p-5">
              <h2 class="text-xl font-black tracking-tight">Clue log</h2>
              <div class="mt-4 space-y-3">
                {#each clueLog.slice().reverse() as clue (`${clue.round}-${clue.team}-${clue.status}`)}
                  <article class="rounded-2xl border border-slate-700 bg-slate-950/80 px-4 py-3">
                    <div class="flex items-center justify-between gap-3">
                      <span class={['rounded-full px-2.5 py-1 text-xs font-black capitalize', clue.team === 'blue' ? 'bg-blue-300 text-blue-950' : 'bg-red-300 text-red-950']}>{clue.team}</span>
                      <span class="text-xs font-bold uppercase tracking-[0.16em] text-slate-500">{clue.status}</span>
                    </div>
                    <p class="mt-2 font-black">{clue.text} · {formatClueNumber(clue.number)}</p>
                    <p class="text-xs text-slate-500">Round {clue.round} · {clue.guesses} guesses</p>
                  </article>
                {:else}
                  <p class="text-sm text-slate-400">No clues submitted yet.</p>
                {/each}
              </div>
            </section>
          </aside>
        </section>
      {/if}
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


{#snippet ChatPanel(messages: ChatMessage[], draft: string, currentPlayer: LobbyPlayer | undefined, onsend: () => void)}
  <section class="rounded-[2rem] border border-slate-700/70 bg-slate-900/80 p-5">
    <div class="flex items-center justify-between gap-3">
      <h2 class="text-xl font-black tracking-tight">Chat</h2>
      {#if !currentPlayer}
        <span class="rounded-full border border-slate-600 px-2.5 py-1 text-xs font-bold text-slate-300">read-only spectator</span>
      {/if}
    </div>
    <div class="mt-4 max-h-72 space-y-3 overflow-y-auto pr-1">
      {#each messages as message (message.id)}
        <article class="rounded-2xl border border-slate-700 bg-slate-950/80 px-4 py-3">
          <p class="text-xs font-black text-emerald-200">{message.displayName || 'Player'}</p>
          <p class="mt-1 break-words text-sm leading-6 text-slate-100">{message.body}</p>
        </article>
      {:else}
        <p class="text-sm text-slate-400">No chat messages yet.</p>
      {/each}
    </div>
    <div class="mt-4 grid grid-cols-[1fr_auto] gap-2">
      <input
        class="rounded-2xl border border-slate-700 bg-slate-950 px-4 py-3 text-slate-50 outline-none ring-emerald-300 transition focus:ring-2 disabled:opacity-60"
        bind:value={chatDraft}
        maxlength="1000"
        placeholder={currentPlayer ? 'Message the room' : 'Spectators are read-only'}
        disabled={!currentPlayer}
        onkeydown={(event) => {
          if (event.key === 'Enter') onsend();
        }}
      />
      <button class="rounded-2xl bg-emerald-300 px-4 py-3 font-black text-slate-950 transition hover:bg-emerald-200 disabled:opacity-50" disabled={!currentPlayer || !draft.trim()} onclick={onsend}>Send</button>
    </div>
  </section>
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
