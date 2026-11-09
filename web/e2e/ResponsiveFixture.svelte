<script lang="ts">
  import BottomControls from '../src/lib/BottomControls.svelte';
  import ModSettings from '../src/lib/ModSettings.svelte';
  import { defaultSettings, type Settings } from '../src/lib/api';
  import { modSettingsShellClasses, roomMainClasses } from '../src/lib/gameplay';
  import type { LobbyPlayer } from '../src/lib/lobby';

  const params = new URLSearchParams(window.location.search);
  const requestedMode = params.get('mode');
  const mode = requestedMode === 'polarity' || requestedMode === 'monality' ? requestedMode : 'unity';
  const team = mode === 'unity' ? 'unity' : mode === 'monality' ? 'monality' : 'blue';
  const isSpy = params.get('role') !== 'guesser';
  let settings = $state<Settings>({ ...defaultSettings, mode });
  let clueText = $state('');
  let clueNumber = $state('');
  let notice = $state('');
  let boardView = $state<'active' | 'own' | 'previous'>('active');
  let spyView = $state(true);
  let open = $state(true);
  const players: LobbyPlayer[] = [
    { id: 'spy', displayName: 'Long player name for narrow screens', team, spymaster: true, representative: false, mod: true },
    { id: 'guesser', displayName: 'Another player', team, spymaster: false, representative: true, mod: false },
  ];
  const noop = () => {};
</script>

<main class={roomMainClasses()}>
  <div class="mx-auto w-full max-w-7xl px-2 py-4 sm:px-8 sm:py-6">
    <p role="status">{notice}</p>
    {#if params.get('surface') === 'settings'}
      <section class="grid min-w-0 gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(0,24rem)]">
        <div class="min-w-0">Players</div>
        <aside class="min-w-0 space-y-6">
          <ModSettings bind:settings hostControls={true} wordpacks={[
            { id: 'english', label: 'English', wordCount: 400 },
            { id: 'long', label: 'Fictional characters and cartoons with a very long title', wordCount: 234 },
          ]} pictures={[]} pictureCatalogAvailable={false} phase="lobby"
            canRandomizeTeams={true} {open} onSave={() => notice = 'Settings saved'}
            onToggleOpen={() => open = !open} onRandomizeTeams={noop} onShuffleRoles={noop}
            onResetClue={noop} onRestartMatch={noop}
            profiles={[{ id: 'long', name: 'A very long saved settings profile name that should never stretch the phone viewport', source: 'local', settings }]} />
        </aside>
      </section>
    {:else}
      <div class={modSettingsShellClasses()}>Board content</div>
      <BottomControls phase="active" currentTeam={team} {mode}
        activeBoardOwner="spy" hasPreviousBoard={true} {boardView}
        currentClue={params.has('clue') ? { text: 'A very long clue in the current turn', number: { kind: 'numeric', value: 2 }, team, round: 1, guesses: 0, status: 'active' } : null}
        role={{ kind: isSpy ? 'spymaster' : 'guesser', activeGuesser: !isSpy, team, player: players[isSpy ? 0 : 1] }}
        cluePermission={{ allowed: isSpy, reason: '' }} bind:clueText bind:clueNumber
        clueProblem={clueText && clueNumber ? '' : 'Enter clue and count'} passProblem="" {settings} {players}
        hostControls={true} compactSwitchUnitySpymaster={params.has('compact')} spymasterViewActive={spyView} onToggleView={() => spyView = !spyView}
        onSetSpyView={(value) => spyView = value} onSetBoardView={(value) => boardView = value}
        onSwitchUnitySpymaster={() => notice = 'Switched spy'} onNavigate={(value) => notice = value}
        onSubmitClue={() => notice = 'Clue submitted'} onPassTurn={() => notice = 'Turn passed'} />
    {/if}
  </div>
</main>
