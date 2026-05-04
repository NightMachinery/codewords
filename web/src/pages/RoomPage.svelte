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
    cardWordTextClasses,
    clueSubmitProblem,
    defaultGameplayPreferences,
    findViewerPlayer,
    formatClueNumber,
    readGameplayPreferences,
    shouldAutoJoinRoom,
    viewerRole,
    writeGameplayPreferences,
    toTitleCase,
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

  import PlayerList from '../lib/PlayerList.svelte';
  import ChatSidebar from '../lib/ChatSidebar.svelte';
  import BottomControls from '../lib/BottomControls.svelte';
  import ModSettings from '../lib/ModSettings.svelte';

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
  let currentTeam = $state<'blue' | 'red' | 'observers' | ''>('');
  let winner = $state<'blue' | 'red' | ''>('');
  let cards = $state<GameplayCard[]>([]);
  let lastSelected = $state<LastSelected | null>(null);
  let remainingCounts = $state<RemainingCounts>({ blue: 0, red: 0, civilian: 0, black: 0 });
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
  let cueNotice = $state('');
  let socket: RoomSocket | null = null;
  let sawSnapshot = false;
  let lastActionId = 0;
  let lastClueSignature = '';
  let spymasterViewActive = $state(true);

  let buckets = $derived(playerBuckets(players));
  let sortedCards = $derived.by(() => {
    const list = cards.map((c, i) => ({ ...c, originalIndex: i }));
    if (cardMode !== 'mixed' || !settings.mixedImageOrderFirst) return list;
    return list.sort((a, b) => {
      if (a.contentType === 'image' && b.contentType !== 'image') return -1;
      if (a.contentType !== 'image' && b.contentType === 'image') return 1;
      return 0;
    });
  });
  let startState = $derived(startReadiness(players));
  let hostControls = $derived(canManageLobby(viewer));
  let currentPlayer = $derived(findViewerPlayer(players, viewer));
  let needsName = $derived(Boolean(credentialMode === 'auth' && !displayName && roomStatus === 'lobby'));
  let role = $derived(viewerRole(players, viewer, currentTeam as any, phase));
  let cluePermission = $derived(canSubmitClue(players, viewer, currentTeam as any, phase));
  let clueNumberParsed = $derived(clueNumber === '∞' ? { kind: 'infinity' } : { kind: 'numeric', value: parseInt(clueNumber, 10) });
  let clueProblem = $derived(clueSubmitProblem(clueText, clueNumberParsed as any, settings));
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
      const nextActionId = message.snapshot.actionId ?? 0;
      const nextClueSignature = clueSignature(message.snapshot.clueLog ?? []);
      if (sawSnapshot && nextActionId > lastActionId) {
        emitCue('cardChoice', 'A card was revealed.');
      }
      if (sawSnapshot && nextClueSignature && nextClueSignature !== lastClueSignature) {
        emitCue('clue', 'New clue received.');
      }
      sawSnapshot = true;
      lastActionId = nextActionId;
      lastClueSignature = nextClueSignature;
      players = message.snapshot.players;
      settings = { ...defaultSettings, ...message.snapshot.settings };
      viewer = message.snapshot.viewer;
      phase = message.snapshot.phase;
      currentTeam = message.snapshot.currentTeam as any;
      winner = message.snapshot.winner as any;
      cards = message.snapshot.cards ?? [];
      lastSelected = message.snapshot.lastSelected ?? null;
      remainingCounts = message.snapshot.remainingCounts ?? { blue: 0, red: 0, civilian: 0, black: 0 };
      clueLog = message.snapshot.clueLog ?? [];
    }
    if (message.type === 'chatMessage') {
      chatMessages = [...chatMessages, message.message].slice(-50);
      emitCue('chat', 'New chat message.');
    }
    if (message.type === 'error') {
      error = message.message;
    }
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

  function emitCue(kind: 'chat' | 'cardChoice' | 'clue', notice: string) {
    const soundEnabled = kind === 'chat' ? preferences.chatSound : kind === 'cardChoice' ? preferences.cardChoiceSound : preferences.clueSound;
    const visualEnabled = kind === 'chat' ? preferences.chatVisualCue : kind === 'cardChoice' ? preferences.cardChoiceVisualCue : preferences.clueVisualCue;
    if (visualEnabled) {
      cueNotice = notice;
      window.setTimeout(() => {
        if (cueNotice === notice) cueNotice = '';
      }, 2400);
    }
    if (soundEnabled) {
      playCue();
    }
  }

  function playCue() {
    try {
      const AudioContextClass = window.AudioContext || (window as unknown as { webkitAudioContext?: typeof AudioContext }).webkitAudioContext;
      if (!AudioContextClass) return;
      const ctx = new AudioContextClass();
      const oscillator = ctx.createOscillator();
      const gain = ctx.createGain();
      oscillator.frequency.value = 660;
      gain.gain.value = 0.035;
      oscillator.connect(gain);
      gain.connect(ctx.destination);
      oscillator.start();
      oscillator.stop(ctx.currentTime + 0.08);
      oscillator.onended = () => void ctx.close();
    } catch {
      // Browsers may block audio until a user gesture; local visual cues still work.
    }
  }

  function clueSignature(entries: ClueEntry[]) {
    const latest = entries.at(-1);
    return latest ? `${latest.round}:${latest.team}:${latest.status}:${latest.text}:${formatClueNumber(latest.number)}:${latest.guesses}` : '';
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

  function shuffleRoles() {
    socket?.send({ type: 'shuffleRoles' });
  }

  function resetClue() {
    socket?.send({ type: 'resetClue' });
  }

  function restartMatch() {
    socket?.send({ type: 'restartMatch' });
  }
</script>

<main class="min-h-screen w-full overflow-x-hidden bg-[oklch(14%_0.018_255)] text-slate-100 pb-32 pr-12">
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
            {phase === 'lobby' ? 'Gather teams, choose roles, then start.' : phase === 'game_over' ? `${winner || 'A team'} wins the board.` : ''}
          </h1>
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
          </div>
        </aside>
      </header>

      {#if phase === 'lobby'}
      <section class="grid gap-6 lg:grid-cols-[1fr_24rem]">
        <div class="space-y-6">
          <PlayerList 
            players={players} 
            viewer={viewer} 
            hostControls={hostControls} 
            roomHostId={roomHostId}
            onAssignTeam={(id, team) => socket?.send({ type: 'assignTeam', playerId: id, team })}
            onToggleSpymaster={(id) => socket?.send({ type: 'toggleSpymaster', playerId: id })}
            onToggleRepresentative={(id) => socket?.send({ type: 'toggleRepresentative', playerId: id })}
            onToggleMod={(id) => socket?.send({ type: 'toggleMod', playerId: id })}
          />
        </div>

        <aside class="space-y-6">
          <ModSettings 
            bind:settings={settings}
            hostControls={hostControls}
            wordpacks={wordpacks}
            pictures={pictures}
            pictureCatalogAvailable={pictureCatalogAvailable}
            onSave={() => socket?.send({ type: 'updateSettings', settings })}
            onShuffleRoles={shuffleRoles}
            onResetClue={resetClue}
            onRestartMatch={restartMatch}
          />

          <section class="rounded-[2rem] border border-slate-700/70 bg-slate-900/80 p-5">
            <h2 class="text-xl font-black tracking-tight">Start match</h2>
            <p class="mt-3 text-sm leading-6 text-slate-300">{startState.ready ? 'The lobby is ready.' : startState.reason}</p>
            <button
              class="mt-5 w-full rounded-2xl bg-emerald-300 px-5 py-3 font-black text-slate-950 transition hover:bg-emerald-200 disabled:cursor-not-allowed disabled:opacity-50"
              disabled={!hostControls || !startState.ready}
              onclick={() => socket?.send({ type: 'startGame' })}
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
                  <div class="flex items-center gap-3">
                    <h2 class="text-2xl font-black tracking-tight">Code grid</h2>
                  </div>
                </div>
                <div class="flex flex-wrap gap-2 text-[10px] font-black uppercase tracking-widest">
                  <span class="rounded-full border border-blue-300/40 bg-blue-400/15 px-3 py-1.5 text-blue-100" style={settings.customColorBlue ? `border-color: ${settings.customColorBlue}66; background-color: ${settings.customColorBlue}26; color: ${settings.customColorBlue}` : ''}>Blue {remainingCounts.blue}</span>
                  <span class="rounded-full border border-red-300/40 bg-red-400/15 px-3 py-1.5 text-red-100" style={settings.customColorRed ? `border-color: ${settings.customColorRed}66; background-color: ${settings.customColorRed}26; color: ${settings.customColorRed}` : ''}>Red {remainingCounts.red}</span>
                  <span class="rounded-full border border-amber-200/40 bg-amber-200/10 px-3 py-1.5 text-amber-100">Civilian {remainingCounts.civilian}</span>
                  <span class="rounded-full border border-zinc-500/40 bg-zinc-950 px-3 py-1.5 text-zinc-100">Assassin {remainingCounts.black}</span>
                </div>
              </div>

              <div class="grid gap-2 md:gap-3" style={`grid-template-columns: repeat(${preferences.cardsPerRow}, minmax(0, 1fr));`}>
                {#each sortedCards as card, index (`${card.word ?? card.imageId ?? 'card'}-${card.originalIndex}`)}
                  {@const showHiddenColor = role.canSeeHiddenColors && (role.kind !== 'spymaster' || spymasterViewActive)}
                  {@const revealedStyle = (role.kind === 'spymaster' && spymasterViewActive) ? preferences.spymasterRevealedStyle : 'normal'}
                  {@const view = cardViewState(card, card.originalIndex, showHiddenColor, lastSelected, revealedStyle)}
                  {@const customColor = card.color === 'blue' ? settings.customColorBlue : card.color === 'red' ? settings.customColorRed : ''}
                  <button
                    class={['group relative overflow-hidden rounded-xl border p-1 text-left shadow-xl shadow-slate-950/25 transition duration-200 hover:-translate-y-0.5 disabled:cursor-not-allowed disabled:hover:translate-y-0', card.contentType === 'image' ? 'aspect-[2/3]' : 'min-h-24 sm:min-h-32', view.classes, !role.activeGuesser || card.revealed || phase !== 'active' ? 'disabled:opacity-80' : ''].join(' ')}
                    style={view.visibleColor !== 'hidden' && customColor ? `border-color: ${customColor}B3; background-color: ${customColor}40; color: white` : ''}
                    disabled={Boolean(guessDisabledReason(card))}
                    title={guessDisabledReason(card) || `Reveal ${cardContentLabel(card)}`}
                    onclick={() => guessCard(card.originalIndex, card)}
                  >
                    {#if card.contentType === 'image'}
                      <img class="h-full w-full rounded-lg object-cover" src={cardImageUrl(card)} alt="Card illustration" loading="lazy" />
                    {:else}
                      <div class="flex h-full items-center justify-center p-2">
                        <span class={cardWordTextClasses(card.word)}>{toTitleCase(card.word) || 'Card'}</span>
                      </div>
                    {/if}
                  </button>
                {:else}
                  <p class="col-span-full rounded-2xl border border-slate-700 bg-slate-950 p-6 text-slate-300">Waiting for the board snapshot...</p>
                {/each}
              </div>
            </section>
          </div>

          <aside class="space-y-6 pb-32">
            <ModSettings 
              bind:settings={settings}
              hostControls={hostControls}
              wordpacks={wordpacks}
              pictures={pictures}
              pictureCatalogAvailable={pictureCatalogAvailable}
              onSave={() => socket?.send({ type: 'updateSettings', settings })}
              onShuffleRoles={shuffleRoles}
              onResetClue={resetClue}
              onRestartMatch={restartMatch}
            />

            <PlayerList 
              players={players} 
              viewer={viewer} 
              hostControls={hostControls} 
              roomHostId={roomHostId}
              onAssignTeam={(id, team) => socket?.send({ type: 'assignTeam', playerId: id, team })}
              onToggleSpymaster={(id) => socket?.send({ type: 'toggleSpymaster', playerId: id })}
              onToggleRepresentative={(id) => socket?.send({ type: 'toggleRepresentative', playerId: id })}
              onToggleMod={(id) => socket?.send({ type: 'toggleMod', playerId: id })}
            />

            <section class="rounded-[2rem] border border-slate-700/70 bg-slate-900/80 p-5">
              <h2 class="text-xl font-black tracking-tight">Clue log</h2>
              <div class="mt-4 space-y-3">
                {#each clueLog.slice().reverse() as clue (`${clue.round}-${clue.team}-${clue.status}`)}
                  <article class="rounded-2xl border border-slate-700 bg-slate-950/80 px-4 py-3">
                    <div class="flex items-center justify-between gap-3">
                      <span class={['rounded-full px-2.5 py-1 text-xs font-black capitalize', clue.team === 'blue' ? 'bg-blue-300 text-blue-950' : 'bg-red-300 text-red-950']} style={clue.team === 'blue' && settings.customColorBlue ? `background-color: ${settings.customColorBlue}; color: white` : clue.team === 'red' && settings.customColorRed ? `background-color: ${settings.customColorRed}; color: white` : ''}>{clue.team}</span>
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

            <section class="rounded-[2rem] border border-slate-700/70 bg-slate-900/80 p-5">
              <h2 class="text-xl font-black tracking-tight">Local Options</h2>
              <label class="mt-4 flex items-center gap-3 rounded-2xl border border-slate-700 bg-slate-950 px-4 py-3 cursor-pointer">
                <input type="checkbox" checked={preferences.confirmGuesses} onchange={(event) => updatePreferences({ confirmGuesses: event.currentTarget.checked })} />
                <span class="text-sm text-slate-200">Confirm before revealing a card</span>
              </label>
              <label class="mt-3 flex items-center gap-3 rounded-2xl border border-slate-700 bg-slate-950 px-4 py-3 cursor-pointer">
                <input type="checkbox" checked={preferences.confirmPasses} onchange={(event) => updatePreferences({ confirmPasses: event.currentTarget.checked })} />
                <span class="text-sm text-slate-200">Confirm before passing</span>
              </label>
              <label class="mt-3 block rounded-2xl border border-slate-700 bg-slate-950 px-4 py-3">
                <span class="text-sm text-slate-200 font-bold">Cards per row: {preferences.cardsPerRow}</span>
                <input class="mt-2 w-full accent-emerald-300" type="range" min="1" max="13" value={preferences.cardsPerRow} oninput={(event) => updatePreferences({ cardsPerRow: Number.parseInt(event.currentTarget.value, 10) })} />
              </label>
              <label class="mt-3 block rounded-2xl border border-slate-700 bg-slate-950 px-4 py-3">
                <span class="text-sm text-slate-200 font-bold">Spymaster view style</span>
                <select class="mt-2 w-full rounded-xl border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-slate-50" bind:value={preferences.spymasterRevealedStyle} onchange={(event) => updatePreferences({ spymasterRevealedStyle: event.currentTarget.value as any })}>
                  <option value="greyed">Greyed</option>
                  <option value="invisible">Invisible</option>
                  <option value="omitted">Omitted</option>
                </select>
              </label>
              <div class="mt-3 grid gap-2 text-sm text-slate-200">
                <label class="flex items-center gap-3 rounded-2xl border border-slate-700 bg-slate-950 px-4 py-3 cursor-pointer">
                  <input type="checkbox" checked={preferences.chatSound} onchange={(event) => updatePreferences({ chatSound: event.currentTarget.checked })} />
                  Chat sound
                </label>
                <label class="flex items-center gap-3 rounded-2xl border border-slate-700 bg-slate-950 px-4 py-3 cursor-pointer">
                  <input type="checkbox" checked={preferences.chatVisualCue} onchange={(event) => updatePreferences({ chatVisualCue: event.currentTarget.checked })} />
                  Chat visual cue
                </label>
                <label class="flex items-center gap-3 rounded-2xl border border-slate-700 bg-slate-950 px-4 py-3 cursor-pointer">
                  <input type="checkbox" checked={preferences.cardChoiceSound} onchange={(event) => updatePreferences({ cardChoiceSound: event.currentTarget.checked })} />
                  Card-choice sound
                </label>
                <label class="flex items-center gap-3 rounded-2xl border border-slate-700 bg-slate-950 px-4 py-3 cursor-pointer">
                  <input type="checkbox" checked={preferences.cardChoiceVisualCue} onchange={(event) => updatePreferences({ cardChoiceVisualCue: event.currentTarget.checked })} />
                  Card-choice visual cue
                </label>
                <label class="flex items-center gap-3 rounded-2xl border border-slate-700 bg-slate-950 px-4 py-3 cursor-pointer">
                  <input type="checkbox" checked={preferences.clueSound} onchange={(event) => updatePreferences({ clueSound: event.currentTarget.checked })} />
                  Incoming clue sound
                </label>
                <label class="flex items-center gap-3 rounded-2xl border border-slate-700 bg-slate-950 px-4 py-3 cursor-pointer">
                  <input type="checkbox" checked={preferences.clueVisualCue} onchange={(event) => updatePreferences({ clueVisualCue: event.currentTarget.checked })} />
                  Incoming clue visual cue
                </label>
              </div>
            </section>
          </aside>
        </section>
      {/if}
    {/if}

    <ChatSidebar 
      messages={chatMessages} 
      bind:draft={chatDraft} 
      canChat={Boolean(currentPlayer && (currentPlayer.team !== 'observers' || settings.observerChatEnabled))}
      onSend={sendChat}
    />

    {#if phase === 'active'}
      <BottomControls 
        phase={phase}
        currentTeam={currentTeam as any}
        currentClue={currentClue}
        role={role}
        cluePermission={cluePermission}
        bind:clueText={clueText}
        bind:clueNumber={clueNumber}
        clueProblem={clueProblem}
        guessProblem={guessProblem}
        passProblem={passProblem}
        settings={settings}
        spymasterViewActive={spymasterViewActive}
        onToggleView={() => (spymasterViewActive = !spymasterViewActive)}
        onSubmitClue={submitClue}
        onPassTurn={passTurn}
      />
    {/if}

    {#if error}
      <p class="fixed top-20 left-1/2 z-50 -translate-x-1/2 rounded-2xl border border-red-400/40 bg-red-400/90 backdrop-blur-md px-6 py-4 text-sm font-bold text-red-50 shadow-2xl">
        {error}
        <button class="ml-4 opacity-70 hover:opacity-100" onclick={() => error = ''}>✕</button>
      </p>
    {/if}
    {#if cueNotice}
      <p class="fixed bottom-24 left-1/2 z-50 -translate-x-1/2 rounded-full border border-emerald-200/60 bg-emerald-300 px-5 py-3 text-sm font-black text-slate-950 shadow-2xl shadow-emerald-950/40">
        {cueNotice}
      </p>
    {/if}
  </div>
</main>
