<script lang="ts">
  import { onDestroy, onMount } from 'svelte';

  import { api, defaultSettings, type ChatMessage, type PictureAsset, type Settings, type Viewer, type Wordpack } from '../lib/api';
  import { copyText } from '../lib/clipboard';
  import {
    activeMatchLayoutClasses,
    canSubmitClue,
    cardAspectRatioClasses,
    boardGridContainerClasses,
    boardGridClasses,
    boardGridStyle,
    cardContentLabel,
    cardImageUrl,
    cardModeFromImageCount,
    cardViewState,
    cardWordTextClasses,
    cardWordTextSegments,
    clueLogKey,
    clueNumberFromInput,
    clueSubmitProblem,
    chatCueNotice,
    chatToggleEventName,
    defaultGameplayPreferences,
    displayCards,
    displayTeamName,
    endGameOutcome,
    buildMemoryCaptureModel,
    findViewerPlayer,
    formatClueNumber,
    imageCardGridStyle,
    lobbyStartPanelClasses,
    pressableButtonClasses,
    normalizeLobbySettingsForSave,
    readPanelPreferences,
    readGameplayPreferences,
    roomMainClasses,
    shouldAutoJoinRoom,
    shouldCueChatMessage,
    shouldCueCardReveal,
    viewerRole,
    writePanelPreferences,
    writeGameplayPreferences,
    toTitleCase,
    hexWithAlpha,
    teamColor,
    type ClueEntry,
    type GameplayCard,
    type GameplayPreferences,
    type ImageCardScale,
    type LastSelected,
    type PanelPreferences,
    type EndGameOutcome,
    type RemainingCounts,
  } from '../lib/gameplay';
  import { downloadMemoryCapture } from '../lib/memoryCapture';
  import { getOrCreateAuthToken, resolveSessionCredential, type SessionCredential } from '../lib/identity';
  import { canManageLobby, playerBuckets, startReadiness, type LobbyPlayer } from '../lib/lobby';
  import { RoomSocket, type BoardLayoutPreferences, type RoomSocketMessage } from '../lib/realtime';
  import { roomIdFromPath, roomPath, websocketRoomUrl } from '../lib/routes';

  import PlayerList from '../lib/PlayerList.svelte';
  import ChatSidebar from '../lib/ChatSidebar.svelte';
  import BottomControls from '../lib/BottomControls.svelte';
  import FitCardWord from '../lib/FitCardWord.svelte';
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
  let panelPreferences = $state<PanelPreferences>({ modSettingsOpen: true, localOptionsOpen: true });
  let loading = $state(true);
  let savingName = $state(false);
  let connection = $state('disconnected');
  let error = $state('');
  let copyStatus = $state('');
  let migrateUrl = $state('');
  let cueNotice = $state('');
  let endGameCue = $state<{ outcome: EndGameOutcome; team: 'blue' | 'red'; text: string } | null>(null);
  let captureStatus = $state('');
  let captureBusy = $state(false);
  let forceBoardLayoutPending = $state(false);
  let socket: RoomSocket | null = null;
  let sawSnapshot = false;
  let previousCardsForCue: GameplayCard[] = [];
  let lastClueSignature = '';
  let spymasterViewActive = $state(true);

  let buckets = $derived(playerBuckets(players));
  let cardMode = $derived(cardModeFromImageCount(settings.imageCardCount ?? 0, settings.totalCards ?? 25));
  let sortedCards = $derived(displayCards(cards, cardMode, settings.mixedImageOrderFirst));
  let canRandomizeTeams = $derived(players.filter((player) => player.team !== 'observers').length >= 2);
  let startState = $derived(startReadiness(players));
  let hostControls = $derived(canManageLobby(viewer));
  let currentPlayer = $derived(findViewerPlayer(players, viewer));
  let needsName = $derived(Boolean(credentialMode === 'auth' && !displayName && roomStatus === 'lobby'));
  let role = $derived(viewerRole(players, viewer, currentTeam as any, phase));
  let activeTeamHasRepresentative = $derived(players.some((player) => player.team === currentTeam && player.representative));
  let cluePermission = $derived(canSubmitClue(players, viewer, currentTeam as any, phase, settings));
  let clueNumberParsed = $derived(clueNumberFromInput(clueNumber));
  let clueProblem = $derived(clueSubmitProblem(clueText, clueNumberParsed as any, settings));
  let currentClue = $derived(clueLog.slice().reverse().find((entry) => entry.status === 'active') ?? null);
  let guessProblem = $derived(guessDisabledReason());
  let passProblem = $derived(passDisabledReason());
  let activeColumns = $derived(preferences.boardColumnsDesktop);
  let mobileColumns = $derived(preferences.boardColumnsMobile);

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
      panelPreferences = readPanelPreferences(localStorage);
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
      rememberCreatorSettings();
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
      const nextClueSignature = clueSignature(message.snapshot.clueLog ?? []);
      const nextCards = message.snapshot.cards ?? [];
      const enteredGameOver = sawSnapshot && phase !== 'game_over' && message.snapshot.phase === 'game_over' && Boolean(message.snapshot.winner);
      if (sawSnapshot && shouldCueCardReveal(previousCardsForCue, nextCards)) {
        emitCue('cardChoice', 'A card was revealed.');
      }
      if (sawSnapshot && nextClueSignature && nextClueSignature !== lastClueSignature) {
        emitCue('clue', 'New clue received.');
      }
      sawSnapshot = true;
      previousCardsForCue = nextCards;
      lastClueSignature = nextClueSignature;
      players = message.snapshot.players;
      settings = { ...defaultSettings, ...message.snapshot.settings };
      rememberCreatorSettings();
      viewer = message.snapshot.viewer;
      phase = message.snapshot.phase;
      currentTeam = message.snapshot.currentTeam as any;
      winner = message.snapshot.winner as any;
      cards = message.snapshot.cards ?? [];
      lastSelected = message.snapshot.lastSelected ?? null;
      remainingCounts = message.snapshot.remainingCounts ?? { blue: 0, red: 0, civilian: 0, black: 0 };
      clueLog = message.snapshot.clueLog ?? [];
      if (enteredGameOver) {
        emitEndGameCue(message.snapshot.winner as 'blue' | 'red', message.snapshot.viewer, message.snapshot.players);
      }
    }
    if (message.type === 'chatMessage') {
      chatMessages = [...chatMessages, message.message].slice(-50);
      if (shouldCueChatMessage(viewer, message.message)) {
        emitCue('chat', chatCueNotice(message.message));
      }
    }
    if (message.type === 'boardLayoutForced') {
      applyBoardLayoutPreferences(message.preferences);
      if (message.by === viewer?.userId) {
        forceBoardLayoutPending = false;
        showToast('Board layout options sent to all players.');
      } else {
        error = 'Board layout options were updated by a moderator.';
      }
    }
    if (message.type === 'error') {
      forceBoardLayoutPending = false;
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
      return `Only the ${displayTeamName(currentTeam, settings)} guesser can reveal cards.`;
    }
    if (settings.enforceClueGuessLimit && (!currentClue || currentClue.number.kind === 'blank')) return 'Wait for a numbered clue first.';
    if (card?.revealed) return 'That card is already revealed.';
    return '';
  }

  function passDisabledReason(): string {
    if (phase === 'game_over') return 'The match is over.';
    if (phase !== 'active') return 'The match has not started.';
    if (!role.player) return 'Spectators are read-only.';
    if (!role.activeGuesser) return role.kind === 'spymaster' ? 'Spymasters cannot pass while teammates can.' : `Only the ${displayTeamName(currentTeam, settings)} guesser can pass.`;
    return '';
  }

  function updatePreferences(next: Partial<GameplayPreferences>) {
    preferences = { ...preferences, ...next };
    writeGameplayPreferences(localStorage, preferences);
  }

  function currentBoardLayoutPreferences(): BoardLayoutPreferences {
    return {
      boardColumnsMobile: preferences.boardColumnsMobile,
      boardColumnsDesktop: preferences.boardColumnsDesktop,
      imageCardScale: preferences.imageCardScale,
      strictCardAspectRatios: preferences.strictCardAspectRatios,
    };
  }

  function applyBoardLayoutPreferences(layout: BoardLayoutPreferences) {
    updatePreferences(layout);
  }

  function forceBoardLayoutForRoom() {
    if (!hostControls) {
      error = 'Only moderators can force board layout options.';
      return;
    }
    if (!socket) {
      error = 'Board layout sync is unavailable until the room reconnects.';
      return;
    }
    forceBoardLayoutPending = true;
    socket.send({ type: 'forceBoardLayout', preferences: currentBoardLayoutPreferences() });
  }

  function updatePanelPreferences(next: Partial<PanelPreferences>) {
    panelPreferences = { ...panelPreferences, ...next };
    writePanelPreferences(localStorage, panelPreferences);
  }


  function emitEndGameCue(winningTeam: 'blue' | 'red', snapshotViewer: Viewer | null | undefined, snapshotPlayers: LobbyPlayer[]) {
    const outcome = endGameOutcome(winningTeam, snapshotViewer, snapshotPlayers);
    const teamName = displayTeamName(winningTeam, settings);
    const text = outcome === 'win' ? `${teamName} takes the board.` : outcome === 'loss' ? `${teamName} wins this round.` : `${teamName} wins.`;
    if (preferences.endGameVisualCue) {
      endGameCue = { outcome, team: winningTeam, text };
      window.setTimeout(() => {
        if (endGameCue?.text === text) endGameCue = null;
      }, 4200);
    }
    if (preferences.endGameSound) {
      playEndGameCue(outcome);
    }
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

  function showToast(notice: string) {
    cueNotice = notice;
    window.setTimeout(() => {
      if (cueNotice === notice) cueNotice = '';
    }, 2400);
  }


  function playEndGameCue(outcome: EndGameOutcome) {
    const notes = outcome === 'win' ? [523.25, 659.25, 783.99, 1046.5] : outcome === 'loss' ? [392, 329.63, 261.63] : [440, 554.37, 659.25];
    try {
      const AudioContextClass = window.AudioContext || (window as unknown as { webkitAudioContext?: typeof AudioContext }).webkitAudioContext;
      if (!AudioContextClass) return;
      const ctx = new AudioContextClass();
      notes.forEach((frequency, index) => {
        const oscillator = ctx.createOscillator();
        const gain = ctx.createGain();
        oscillator.type = outcome === 'loss' ? 'triangle' : 'sine';
        oscillator.frequency.value = frequency;
        const start = ctx.currentTime + index * 0.11;
        gain.gain.setValueAtTime(0.0001, start);
        gain.gain.exponentialRampToValueAtTime(outcome === 'loss' ? 0.045 : 0.065, start + 0.025);
        gain.gain.exponentialRampToValueAtTime(0.0001, start + 0.26);
        oscillator.connect(gain);
        gain.connect(ctx.destination);
        oscillator.start(start);
        oscillator.stop(start + 0.28);
      });
      window.setTimeout(() => void ctx.close(), 900);
    } catch {
      // Browsers may block audio until a user gesture; local visual cues still work.
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
      clearCopyStatusSoon(copyStatus);
  }

  async function copyMigrateLink() {
    error = '';
    try {
      const link = await api.createMigrateLink(roomId, authToken);
      migrateUrl = link.migrateUrl;
      const result = await copyText(link.migrateUrl);
      copyStatus = result.ok ? 'Migrate-device link copied.' : link.migrateUrl;
      clearCopyStatusSoon(copyStatus);
    } catch (err) {
      error = err instanceof Error ? err.message : 'Could not create a migrate link.';
    }
  }

  function shuffleRoles() {
    socket?.send({ type: 'shuffleRoles' });
  }

  function randomizeTeams() {
    socket?.send({ type: 'randomizeTeams' });
  }

  function imageSelectionBorder(color: string) {
    if (color === 'blue') return teamColor('blue', settings) || '#93c5fd';
    if (color === 'red') return teamColor('red', settings) || '#fca5a5';
    if (color === 'black') return '#d4d4d8';
    if (color === 'civilian') return '#fde68a';
    return '#e5e7eb';
  }

  function resetClue() {
    socket?.send({ type: 'resetClue' });
  }

  function restartMatch() {
    if (window.confirm('Restart this match and return everyone to the lobby?')) {
      socket?.send({ type: 'restartMatch' });
    }
  }

  function clearCopyStatusSoon(value: string) {
    window.setTimeout(() => {
      if (copyStatus === value) copyStatus = '';
    }, 2400);
  }

  function navigateTo(target: string) {
    if (target === 'chat') {
      window.dispatchEvent(new CustomEvent(chatToggleEventName));
      return;
    }
    document.getElementById(target)?.scrollIntoView({ behavior: 'smooth', block: 'start' });
  }


  async function captureMemory() {
    if (winner !== 'blue' && winner !== 'red') return;
    captureBusy = true;
    captureStatus = '';
    error = '';
    try {
      const model = buildMemoryCaptureModel({ roomId, winner, players, cards, settings });
      await downloadMemoryCapture(model);
      captureStatus = 'Memory image downloaded.';
      clearCaptureStatusSoon(captureStatus);
    } catch (err) {
      error = err instanceof Error ? err.message : 'Could not capture the memory image.';
    } finally {
      captureBusy = false;
    }
  }

  function clearCaptureStatusSoon(value: string) {
    window.setTimeout(() => {
      if (captureStatus === value) captureStatus = '';
    }, 2600);
  }

  function rememberCreatorSettings() {
    if (!viewer?.isHost || !viewer.userId) return;
    const saved = { ...settings, seed: undefined };
    localStorage.setItem(`codewords.creatorSettings.${viewer.userId}`, JSON.stringify(saved));
  }

  function saveRoomSettings() {
    settings = normalizeLobbySettingsForSave(settings);
    socket?.send({ type: 'updateSettings', settings });
  }
</script>

<main class={roomMainClasses()}>
  <div class="mx-auto w-full max-w-7xl px-2 py-4 sm:px-8 sm:py-6">
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
            class={pressableButtonClasses('mt-4 w-full rounded-2xl bg-emerald-300 px-5 py-3 font-black text-slate-950 hover:bg-emerald-200 disabled:opacity-60')}
            disabled={savingName}
            onclick={saveNameAndJoin}
          >
            {savingName ? 'Joining...' : 'Join lobby'}
          </button>
        </div>
      </section>
    {:else}
      {#if phase !== 'active'}
      <header class="grid gap-8 px-3 py-8 sm:px-0 lg:grid-cols-[1fr_22rem] lg:py-16">
        <div class="max-w-5xl">
          <h1 class="max-w-5xl text-5xl font-black leading-[0.96] tracking-[-0.05em] text-slate-50 sm:text-7xl">
            {phase === 'lobby' ? 'Gather teams, choose roles, then start.' : phase === 'game_over' ? `${displayTeamName(winner, settings)} wins the board.` : ''}
          </h1>
        </div>

        <aside class="self-start rounded-[2rem] border border-slate-700/70 bg-slate-900/80 p-5 shadow-2xl shadow-slate-950/40">
          <div class="grid gap-3">
            <button class={pressableButtonClasses('rounded-2xl bg-slate-100 px-5 py-3 font-black text-slate-950 hover:bg-white')} onclick={copyRoomLink}>Copy room link</button>
            {#if currentPlayer}
              <button class={pressableButtonClasses('rounded-2xl border border-slate-600 px-5 py-3 font-bold text-slate-100 hover:border-emerald-300 hover:text-emerald-200')} onclick={copyMigrateLink}>
                Copy migrate-device link
              </button>
            {/if}
            {#if copyStatus}
              <p class="break-all rounded-2xl border border-emerald-300/40 bg-emerald-300/10 px-4 py-3 text-sm text-emerald-100">{copyStatus}</p>
            {/if}
          </div>
        </aside>
      </header>
      {/if}

      {#if phase === 'lobby'}
      <section class="grid min-w-0 gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(0,24rem)]">
        <div class="min-w-0 space-y-6">
          <PlayerList 
            players={players} 
            viewer={viewer} 
            settings={settings}
            hostControls={hostControls} 
            roomHostId={roomHostId}
            onAssignTeam={(id, team) => socket?.send({ type: 'assignTeam', playerId: id, team })}
            onToggleSpymaster={(id) => socket?.send({ type: 'toggleSpymaster', playerId: id })}
            onToggleRepresentative={(id) => socket?.send({ type: 'toggleRepresentative', playerId: id })}
            onToggleMod={(id) => socket?.send({ type: 'toggleMod', playerId: id })}
            onRejoinTeam={(id) => socket?.send({ type: 'rejoinTeam', playerId: id })}
          />
        </div>

        <aside class="min-w-0 space-y-6">
          <ModSettings 
            bind:settings={settings}
            hostControls={hostControls}
            wordpacks={wordpacks}
            pictures={pictures}
            pictureCatalogAvailable={pictureCatalogAvailable}
            onSave={saveRoomSettings}
            phase={phase}
            canRandomizeTeams={canRandomizeTeams}
            open={panelPreferences.modSettingsOpen}
            onToggleOpen={() => updatePanelPreferences({ modSettingsOpen: !panelPreferences.modSettingsOpen })}
            onRandomizeTeams={randomizeTeams}
            onShuffleRoles={shuffleRoles}
            onResetClue={resetClue}
            onRestartMatch={restartMatch}
          />

        </aside>
      </section>
      <section class={lobbyStartPanelClasses()} aria-label="Start match">
        <div class="mx-auto flex max-w-7xl flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <p class="min-w-0 text-sm font-bold text-slate-300">
            {hostControls ? (startState.ready ? 'Lobby ready.' : startState.reason) : 'Waiting for a moderator to start.'}
          </p>
          <button
            class={pressableButtonClasses('w-full rounded-2xl bg-emerald-300 px-5 py-3 text-sm font-black text-slate-950 hover:bg-emerald-200 disabled:cursor-not-allowed disabled:opacity-50 sm:w-auto sm:min-w-40')}
            disabled={!hostControls || !startState.ready}
            onclick={() => socket?.send({ type: 'startGame' })}
          >
            Start match
          </button>
        </div>
      </section>
      {:else}
        <section class={activeMatchLayoutClasses()}>
          <div class="space-y-6">
            {#if phase === 'game_over'}
              <section class={['relative overflow-hidden rounded-[2rem] border p-6 shadow-2xl shadow-slate-950/30', winner === 'blue' ? 'border-blue-300/50 bg-blue-400/15' : 'border-red-300/50 bg-red-400/15']} style={winner ? `border-color: ${hexWithAlpha(teamColor(winner, settings), '80')}; background-color: ${hexWithAlpha(teamColor(winner, settings), '26')};` : ''}>
                <div class="pointer-events-none absolute -right-14 -top-16 h-48 w-48 rounded-full opacity-25 blur-2xl" style={`background-color: ${teamColor(winner, settings)}`}></div>
                <div class="relative flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
                  <div>
                    <p class="text-sm font-black uppercase tracking-[0.25em] text-emerald-200">Game over</p>
                    <h2 class="mt-2 text-4xl font-black tracking-[-0.04em] text-slate-50">{displayTeamName(winner, settings)} wins</h2>
                    <p class="mt-3 max-w-2xl text-slate-300">All card colors are revealed. Save the final board, winner, rivals, and team rosters as a keepsake.</p>
                    {#if captureStatus}
                      <p class="mt-3 text-sm font-bold text-emerald-200">{captureStatus}</p>
                    {/if}
                  </div>
                  <button
                    class="inline-flex items-center justify-center gap-3 rounded-2xl bg-slate-100 px-5 py-3 font-black text-slate-950 shadow-xl shadow-slate-950/30 transition hover:bg-emerald-200 disabled:cursor-wait disabled:opacity-70"
                    type="button"
                    disabled={captureBusy}
                    onclick={captureMemory}
                  >
                    <svg class="h-5 w-5" viewBox="0 0 24 24" aria-hidden="true">
                      <path d="M5 5.5C5 4.1 6.1 3 7.5 3h9C17.9 3 19 4.1 19 5.5v13c0 .8-.9 1.3-1.6.9L12 16.2l-5.4 3.2c-.7.4-1.6-.1-1.6-.9v-13Z" fill="currentColor" opacity="0.2" />
                      <path d="M8 7h8M8 10h5M7 3.75h10A1.25 1.25 0 0 1 18.25 5v12.6l-5.6-3.1a1.3 1.3 0 0 0-1.3 0l-5.6 3.1V5A1.25 1.25 0 0 1 7 3.75Z" fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="1.8" />
                      <path d="M15.5 13.2c1.5-.9 2.4-2.1 2.4-3.5 0-2.4-2.6-4.3-5.9-4.3s-5.9 1.9-5.9 4.3c0 1.4.9 2.6 2.4 3.5" fill="none" stroke="currentColor" stroke-linecap="round" stroke-width="1.4" opacity="0.75" />
                    </svg>
                    {captureBusy ? 'Capturing...' : 'Capture Memory'}
                  </button>
                </div>
              </section>
            {/if}

            <section class="rounded-[1.5rem] border border-slate-700/70 bg-slate-900/70 p-2 shadow-2xl shadow-slate-950/35 sm:rounded-[2rem] sm:p-5">
              <div class="mb-3 flex flex-wrap items-center justify-between gap-3 sm:mb-5">
                <div>
                  <p class="text-xs font-black uppercase tracking-[0.22em] text-slate-400">Board</p>
                </div>
                <div class="flex flex-wrap gap-2 text-[10px] font-black uppercase tracking-widest">
                  <span class="relative isolate rounded-full border border-blue-300/40 bg-blue-400/15 px-3 py-1.5 text-blue-100" style={`border-color: ${hexWithAlpha(teamColor('blue', settings), '66')}; background-color: ${hexWithAlpha(teamColor('blue', settings), currentTeam === 'blue' ? '33' : '26')}; color: ${teamColor('blue', settings)}; ${currentTeam === 'blue' ? `box-shadow: 0 0 0 1px ${hexWithAlpha(teamColor('blue', settings), '55')}, 0 0 28px ${hexWithAlpha(teamColor('blue', settings), '55')};` : ''}`}>{displayTeamName('blue', settings)} {remainingCounts.blue}</span>
                  <span class="relative isolate rounded-full border border-red-300/40 bg-red-400/15 px-3 py-1.5 text-red-100" style={`border-color: ${hexWithAlpha(teamColor('red', settings), '66')}; background-color: ${hexWithAlpha(teamColor('red', settings), currentTeam === 'red' ? '33' : '26')}; color: ${teamColor('red', settings)}; ${currentTeam === 'red' ? `box-shadow: 0 0 0 1px ${hexWithAlpha(teamColor('red', settings), '55')}, 0 0 28px ${hexWithAlpha(teamColor('red', settings), '55')};` : ''}`}>{displayTeamName('red', settings)} {remainingCounts.red}</span>
                  <span class="rounded-full border border-amber-200/40 bg-amber-200/10 px-3 py-1.5 text-amber-100">Civilian {remainingCounts.civilian}</span>
                  <span class="rounded-full border border-zinc-500/40 bg-zinc-950 px-3 py-1.5 text-zinc-100">Assassin {remainingCounts.black}</span>
                </div>
              </div>

              <div class={boardGridContainerClasses()}>
                <div id="board" class={['grid grid-flow-dense gap-2 md:gap-3 [grid-template-columns:repeat(var(--mobile-card-columns),minmax(0,1fr))] md:[grid-template-columns:repeat(var(--card-columns),minmax(0,1fr))]', boardGridClasses()].join(' ')} style={boardGridStyle(mobileColumns, activeColumns)}>
                  {#each sortedCards as card, index (`${card.word ?? card.imageId ?? 'card'}-${card.originalIndex}`)}
                    {@const showHiddenColor = role.canSeeHiddenColors && (role.kind !== 'spymaster' || spymasterViewActive)}
                    {@const revealedStyle = (role.kind === 'spymaster' && spymasterViewActive) ? preferences.spymasterRevealedStyle : 'normal'}
                    {@const view = cardViewState(card, card.originalIndex, showHiddenColor, lastSelected, revealedStyle)}
                    {@const customColor = card.color === 'blue' ? teamColor('blue', settings) : card.color === 'red' ? teamColor('red', settings) : ''}
                    <button
                      class={pressableButtonClasses(['group relative col-span-[var(--card-mobile-col-span)] row-span-[var(--card-mobile-row-span)] md:col-span-[var(--card-col-span)] md:row-span-[var(--card-row-span)] rounded-xl border p-1 text-left duration-200 hover:-translate-y-0.5 disabled:cursor-not-allowed disabled:hover:translate-y-0', cardAspectRatioClasses(card, preferences.strictCardAspectRatios), card.contentType === 'image' ? 'overflow-hidden border-4' : 'overflow-visible', view.classes, !role.activeGuesser || card.revealed || phase !== 'active' ? 'disabled:opacity-80' : ''].join(' '))}
                      style={`${imageCardGridStyle(card, activeColumns, preferences.imageCardScale, mobileColumns)} ${view.visibleColor !== 'hidden' && customColor ? `border-color: ${hexWithAlpha(customColor, 'B3')}; background-color: ${hexWithAlpha(customColor, '40')}; color: white` : ''}`}
                      disabled={Boolean(guessDisabledReason(card))}
                      title={guessDisabledReason(card) || `Reveal ${cardContentLabel(card)}`}
                      onclick={() => guessCard(card.originalIndex, card)}
                    >
                      <span class="absolute left-0 top-0 z-10 rounded-br-lg border-b border-r border-slate-100/20 bg-slate-950/85 px-1.5 py-1 text-[10px] font-black leading-none text-slate-100">
                        #{card.badgeNumber}
                      </span>
                      {#if card.contentType === 'image'}
                        <img class="h-full w-full rounded-lg object-cover" src={cardImageUrl(card)} alt="Card illustration" loading="lazy" />
                        {#if view.isLastSelected}
                          <span class="pointer-events-none absolute inset-1 rounded-lg border-4" style={`border-color: ${imageSelectionBorder(view.visibleColor)}; box-shadow: inset 0 0 0 2px rgba(16, 185, 129, 0.65);`}></span>
                        {/if}
                      {:else}
                        {@const wordSegments = cardWordTextSegments(toTitleCase(card.word) || 'Card')}
                        <FitCardWord segments={wordSegments} classes={cardWordTextClasses(card.word)} />
                      {/if}
                    </button>
                  {:else}
                    <p class="col-span-full rounded-2xl border border-slate-700 bg-slate-950 p-6 text-slate-300">Waiting for the board snapshot...</p>
                  {/each}
                </div>
              </div>
            </section>
          </div>

          <aside class="space-y-6 pb-32">
            <PlayerList 
              players={players} 
              viewer={viewer} 
              settings={settings}
              hostControls={hostControls} 
              roomHostId={roomHostId}
              onAssignTeam={(id, team) => socket?.send({ type: 'assignTeam', playerId: id, team })}
              onToggleSpymaster={(id) => socket?.send({ type: 'toggleSpymaster', playerId: id })}
              onToggleRepresentative={(id) => socket?.send({ type: 'toggleRepresentative', playerId: id })}
              onToggleMod={(id) => socket?.send({ type: 'toggleMod', playerId: id })}
              onRejoinTeam={(id) => socket?.send({ type: 'rejoinTeam', playerId: id })}
            />

            <section id="clues" class="rounded-[2rem] border border-slate-700/70 bg-slate-900/80 p-5">
              <h2 class="text-xl font-black tracking-tight">Clue log</h2>
              <div class="mt-4 space-y-3">
                {#each clueLog.slice().reverse() as clue, index (clueLogKey(clue, index))}
                  <article class="rounded-2xl border border-slate-700 bg-slate-950/80 px-4 py-3">
                    <div class="flex items-center justify-between gap-3">
                      <span class={['rounded-full px-2.5 py-1 text-xs font-black capitalize', clue.team === 'blue' ? 'bg-blue-300 text-blue-950' : 'bg-red-300 text-red-950']} style={`background-color: ${teamColor(clue.team, settings)}; color: white`}>{displayTeamName(clue.team, settings)}</span>
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

            <ModSettings 
              bind:settings={settings}
              hostControls={hostControls}
              wordpacks={wordpacks}
              pictures={pictures}
              pictureCatalogAvailable={pictureCatalogAvailable}
              onSave={saveRoomSettings}
              phase={phase}
              canRandomizeTeams={canRandomizeTeams}
              open={panelPreferences.modSettingsOpen}
              onToggleOpen={() => updatePanelPreferences({ modSettingsOpen: !panelPreferences.modSettingsOpen })}
              onRandomizeTeams={randomizeTeams}
              onShuffleRoles={shuffleRoles}
              onResetClue={resetClue}
              onRestartMatch={restartMatch}
            />

            <section id="local-options" class="rounded-[2rem] border border-slate-700/70 bg-slate-900/80 p-5">
              <button class="group flex w-full items-center gap-3 text-left" type="button" onclick={() => updatePanelPreferences({ localOptionsOpen: !panelPreferences.localOptionsOpen })} aria-expanded={panelPreferences.localOptionsOpen}>
                <span class="grid h-9 w-9 shrink-0 place-items-center rounded-full border border-slate-700 bg-slate-950 text-sm font-black text-slate-300 transition group-hover:border-emerald-300/60 group-hover:text-emerald-200">{panelPreferences.localOptionsOpen ? '−' : '+'}</span>
                <span>
                  <span class="block text-xl font-black tracking-tight">Local Settings</span>
                  <span class="text-xs font-bold uppercase tracking-[0.18em] text-slate-500">{panelPreferences.localOptionsOpen ? 'Open' : 'Closed'}</span>
                </span>
              </button>
              {#if panelPreferences.localOptionsOpen}
                <label class="mt-4 flex items-center gap-3 rounded-2xl border border-slate-700 bg-slate-950 px-4 py-3 cursor-pointer">
                  <input type="checkbox" checked={preferences.confirmGuesses} onchange={(event) => updatePreferences({ confirmGuesses: event.currentTarget.checked })} />
                  <span class="text-sm text-slate-200">Confirm before revealing a card</span>
                </label>
                <label class="mt-3 flex items-center gap-3 rounded-2xl border border-slate-700 bg-slate-950 px-4 py-3 cursor-pointer">
                  <input type="checkbox" checked={preferences.confirmPasses} onchange={(event) => updatePreferences({ confirmPasses: event.currentTarget.checked })} />
                  <span class="text-sm text-slate-200">Confirm before passing</span>
                </label>
                <div class="mt-3 grid gap-3 rounded-2xl border border-slate-700 bg-slate-950 px-4 py-3">
                  <span class="text-sm font-bold text-slate-200">Board layout</span>
                  <label class="text-xs text-slate-400">Mobile columns: {preferences.boardColumnsMobile}<input class="mt-1 w-full accent-emerald-300" type="range" min="1" max="13" value={preferences.boardColumnsMobile} oninput={(event) => updatePreferences({ boardColumnsMobile: Number.parseInt(event.currentTarget.value, 10) })} /></label>
                  <label class="text-xs text-slate-400">Desktop columns: {preferences.boardColumnsDesktop}<input class="mt-1 w-full accent-emerald-300" type="range" min="1" max="13" value={preferences.boardColumnsDesktop} oninput={(event) => updatePreferences({ boardColumnsDesktop: Number.parseInt(event.currentTarget.value, 10) })} /></label>
                  <label class="flex items-center gap-3 rounded-xl border border-slate-800 bg-slate-900/60 px-3 py-2 text-xs text-slate-300 cursor-pointer">
                    <input type="checkbox" checked={preferences.strictCardAspectRatios} onchange={(event) => updatePreferences({ strictCardAspectRatios: event.currentTarget.checked })} />
                    Strictly enforce 4:3 word cards and 2:3 image cards
                  </label>
                  <label class="block text-xs text-slate-400">
                    Image size
                    <select class="mt-1 w-full rounded-xl border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-slate-50" value={String(preferences.imageCardScale)} onchange={(event) => updatePreferences({ imageCardScale: Number.parseInt(event.currentTarget.value, 10) as ImageCardScale })}>
                      <option value="1">Compact, 1×1</option>
                      <option value="2">Tall, 1×2</option>
                      <option value="4">Large, 2×4</option>
                      <option value="8">Poster, 4×8</option>
                    </select>
                  </label>
                  {#if hostControls}
                    <button
                      class={['rounded-xl border px-3 py-2 text-left text-xs font-black uppercase tracking-[0.16em] transition active:translate-y-px disabled:cursor-wait', forceBoardLayoutPending ? 'border-emerald-200 bg-emerald-300 text-slate-950' : 'border-emerald-300/40 bg-emerald-300/10 text-emerald-100 hover:border-emerald-200 hover:bg-emerald-300/20'].join(' ')}
                      type="button"
                      disabled={forceBoardLayoutPending}
                      aria-busy={forceBoardLayoutPending}
                      onclick={forceBoardLayoutForRoom}
                    >
                      {forceBoardLayoutPending ? 'Forcing board layout…' : 'Force these board layout options to all players'}
                    </button>
                  {/if}
                </div>
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
                  <label class="flex items-center gap-3 rounded-2xl border border-slate-700 bg-slate-950 px-4 py-3 cursor-pointer">
                    <input type="checkbox" checked={preferences.endGameSound} onchange={(event) => updatePreferences({ endGameSound: event.currentTarget.checked })} />
                    End-game sound
                  </label>
                  <label class="flex items-center gap-3 rounded-2xl border border-slate-700 bg-slate-950 px-4 py-3 cursor-pointer">
                    <input type="checkbox" checked={preferences.endGameVisualCue} onchange={(event) => updatePreferences({ endGameVisualCue: event.currentTarget.checked })} />
                    End-game visual cue
                  </label>
                </div>
              {/if}
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
        activeTeamHasRepresentative={activeTeamHasRepresentative}
        settings={settings}
        players={players}
        onNavigate={navigateTo}
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
    {#if endGameCue}
      <div class={['pointer-events-none fixed inset-0 z-40 grid place-items-center overflow-hidden', endGameCue.outcome === 'win' ? 'animate-[endgame-pop_4s_ease-out_forwards]' : 'animate-[endgame-fade_4s_ease-out_forwards]'].join(' ')} aria-hidden="true">
        <div class="absolute inset-0" style={`background: radial-gradient(circle at 50% 38%, ${hexWithAlpha(teamColor(endGameCue.team, settings), endGameCue.outcome === 'loss' ? '33' : '66')}, transparent 55%);`}></div>
        <div class={['relative rounded-[2rem] border px-8 py-6 text-center shadow-2xl backdrop-blur-sm', endGameCue.outcome === 'loss' ? 'border-slate-400/30 bg-slate-950/80 text-slate-100' : 'border-emerald-200/60 bg-slate-950/70 text-slate-50'].join(' ')}>
          <p class="text-xs font-black uppercase tracking-[0.28em] text-emerald-200">{endGameCue.outcome === 'win' ? 'Victory' : endGameCue.outcome === 'loss' ? 'Final board' : 'Game over'}</p>
          <p class="mt-2 text-3xl font-black tracking-[-0.04em]">{endGameCue.text}</p>
        </div>
      </div>
    {/if}
    {#if cueNotice}
      <p class="fixed bottom-24 left-1/2 z-50 -translate-x-1/2 rounded-full border border-emerald-200/60 bg-emerald-300 px-5 py-3 text-sm font-black text-slate-950 shadow-2xl shadow-emerald-950/40">
        {cueNotice}
      </p>
    {/if}
  </div>
</main>


<style>
  @keyframes endgame-pop {
    0% { opacity: 0; transform: scale(0.96); }
    12% { opacity: 1; transform: scale(1); }
    78% { opacity: 1; transform: scale(1.01); }
    100% { opacity: 0; transform: scale(1.04); }
  }

  @keyframes endgame-fade {
    0% { opacity: 0; }
    16% { opacity: 1; }
    78% { opacity: 1; }
    100% { opacity: 0; }
  }

  @media (prefers-reduced-motion: reduce) {
    :global(.animate-\[endgame-pop_4s_ease-out_forwards\]),
    :global(.animate-\[endgame-fade_4s_ease-out_forwards\]) {
      animation-name: endgame-fade;
    }
  }
</style>
