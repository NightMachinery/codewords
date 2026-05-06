<script lang="ts">
  import type { PictureAsset, Settings, Wordpack } from './api';
  import { cardModeFromImageCount, colorPickerCtaLabel, displayTeamName, imageCountForMode, isValidHexColor, teamColor } from './gameplay';

  interface Props {
    settings: Settings;
    hostControls: boolean;
    wordpacks: Wordpack[];
    pictures: PictureAsset[];
    pictureCatalogAvailable: boolean;
    phase: 'lobby' | 'active' | 'game_over';
    canRandomizeTeams: boolean;
    open: boolean;
    onSave: () => void;
    onToggleOpen: () => void;
    onRandomizeTeams: () => void;
    onShuffleRoles: () => void;
    onResetClue: () => void;
    onRestartMatch: () => void;
  }

  let {
    settings = $bindable(),
    hostControls,
    wordpacks,
    pictures,
    pictureCatalogAvailable,
    phase,
    canRandomizeTeams,
    open,
    onSave,
    onToggleOpen,
    onRandomizeTeams,
    onShuffleRoles,
    onResetClue,
    onRestartMatch
  }: Props = $props();

  let cardMode = $derived(cardModeFromImageCount(settings.imageCardCount ?? 0));
  let openColorPicker = $state<'blue' | 'red' | null>(null);
  const colorPresets = [
    '#ef4444', '#f97316', '#f59e0b', '#eab308', '#84cc16', '#22c55e', '#10b981', '#14b8a6', '#06b6d4',
    '#0ea5e9', '#3b82f6', '#6366f1', '#8b5cf6', '#a855f7', '#d946ef', '#ec4899', '#f43f5e', '#64748b',
    '#dc2626', '#ea580c', '#d97706', '#ca8a04', '#65a30d', '#16a34a', '#059669', '#0d9488', '#0891b2',
    '#0284c7', '#2563eb', '#4f46e5', '#7c3aed', '#9333ea', '#c026d3', '#db2777', '#e11d48', '#475569',
    '#991b1b', '#9a3412', '#92400e', '#854d0e', '#3f6212', '#166534', '#065f46', '#115e59', '#155e75',
    '#075985', '#1d4ed8', '#3730a3', '#5b21b6', '#6b21a8', '#86198f', '#9d174d', '#9f1239', '#334155',
  ];

  function setCardMode(mode: 'words' | 'images' | 'mixed') {
    settings.imageCardCount = imageCountForMode(mode, settings.imageCardCount);
    onSave();
  }

  function setMixedImageCount(count: string) {
    settings.imageCardCount = Number.parseInt(count, 10);
    onSave();
  }

  function colorInputLabel(team: 'blue' | 'red', color: string | undefined, fallback: string) {
    return `${displayTeamName(team, settings)} color ${color || fallback}`;
  }

  function setTeamColor(team: 'blue' | 'red', color: string) {
    if (team === 'blue') settings.customColorBlue = isValidHexColor(color) ? color : '';
    else settings.customColorRed = isValidHexColor(color) ? color : '';
    onSave();
  }

  function closeColorPicker() {
    openColorPicker = null;
  }

  function colorInputValue(team: 'blue' | 'red') {
    return team === 'blue' ? settings.customColorBlue || '' : settings.customColorRed || '';
  }

  function setColorInputValue(team: 'blue' | 'red', color: string) {
    if (team === 'blue') settings.customColorBlue = color;
    else settings.customColorRed = color;
    onSave();
  }

  function resetTeamColor(team: 'blue' | 'red') {
    if (team === 'blue') settings.customColorBlue = '';
    else settings.customColorRed = '';
    onSave();
  }
</script>

<section id="settings" class="rounded-[2rem] border border-slate-700/70 bg-slate-900/80 p-6 shadow-2xl shadow-slate-950/30">
  <div class="flex items-center justify-between gap-3 mb-6">
    <button class="group flex min-w-0 flex-1 items-center gap-3 text-left" type="button" onclick={onToggleOpen} aria-expanded={open}>
      <span class="grid h-9 w-9 shrink-0 place-items-center rounded-full border border-slate-700 bg-slate-950 text-sm font-black text-slate-300 transition group-hover:border-emerald-300/60 group-hover:text-emerald-200">{open ? '−' : '+'}</span>
      <span>
        <span class="block text-2xl font-black tracking-tight">Mod Settings</span>
        <span class="text-xs font-bold uppercase tracking-[0.18em] text-slate-500">{open ? 'Open' : 'Closed'}</span>
      </span>
    </button>
    <div class="flex items-center gap-2">
      {#if !hostControls}
        <span class="rounded-full bg-slate-800 px-3 py-1 text-[10px] font-black uppercase tracking-widest text-slate-500">Read-only</span>
      {/if}
    </div>
  </div>

  {#if open}
  <fieldset class="space-y-6 disabled:opacity-60" disabled={!hostControls}>
    {#if phase === 'lobby'}
      <!-- Lobby Tools -->
      <div class="space-y-3 rounded-2xl border border-emerald-400/30 bg-emerald-400/10 p-4">
        <span class="text-xs font-black uppercase tracking-widest text-emerald-200">Randomize teams</span>
        <button class="w-full rounded-xl border border-emerald-400/60 bg-emerald-400/10 px-4 py-3 text-sm font-black text-emerald-100 transition hover:bg-emerald-400/20 disabled:cursor-not-allowed disabled:opacity-50" disabled={!canRandomizeTeams} onclick={onRandomizeTeams}>Randomize Teams</button>
        <p class="text-xs leading-5 text-emerald-100/70">Balances non-observer and unassigned players, clears rep roles, and picks one spy for each team.</p>
      </div>
    {/if}

    <!-- Game Rules -->
    <div class="grid gap-4 sm:grid-cols-2">
      <label class="block">
        <span class="text-sm font-bold text-slate-300">Wordpack</span>
        <select class="mt-2 w-full rounded-2xl border border-slate-700 bg-slate-950 px-4 py-3 text-slate-50" bind:value={settings.wordpackId} onchange={onSave}>
          {#each wordpacks as pack (pack.id)}
            <option value={pack.id}>{pack.label} ({pack.wordCount})</option>
          {/each}
        </select>
      </label>
      <label class="block">
        <span class="text-sm font-bold text-slate-300">Black cards (Assassins)</span>
        <input class="mt-2 w-full rounded-2xl border border-slate-700 bg-slate-950 px-4 py-3 text-slate-50" type="number" min="0" max="8" bind:value={settings.blackCards} onchange={onSave} />
      </label>
    </div>

    <!-- Card Content -->
    <div class="rounded-2xl border border-slate-700 bg-slate-950/50 p-5">
      <span class="text-sm font-bold text-slate-300">Card Content</span>
      <div class="mt-4 grid gap-3 sm:grid-cols-3">
        <button class={['rounded-xl border p-3 text-sm font-bold transition', cardMode === 'words' ? 'border-emerald-400 bg-emerald-400/10 text-emerald-100' : 'border-slate-700 hover:border-slate-500']} onclick={() => setCardMode('words')}>Words only</button>
        <button class={['rounded-xl border p-3 text-sm font-bold transition', cardMode === 'images' ? 'border-emerald-400 bg-emerald-400/10 text-emerald-100' : 'border-slate-700 hover:border-slate-500']} disabled={!pictureCatalogAvailable} onclick={() => setCardMode('images')}>Images only</button>
        <button class={['rounded-xl border p-3 text-sm font-bold transition', cardMode === 'mixed' ? 'border-emerald-400 bg-emerald-400/10 text-emerald-100' : 'border-slate-700 hover:border-slate-500']} disabled={!pictureCatalogAvailable} onclick={() => setCardMode('mixed')}>Mixed</button>
      </div>

      {#if cardMode === 'mixed'}
        <label class="mt-5 block">
          <div class="flex justify-between text-xs font-bold text-slate-400">
            <span>Image cards</span>
            <span>{settings.imageCardCount}</span>
          </div>
          <input class="mt-2 w-full accent-emerald-300" type="range" min="1" max="24" value={settings.imageCardCount} oninput={(e) => setMixedImageCount(e.currentTarget.value)} />
        </label>
        <label class="mt-3 flex items-center gap-3">
          <input type="checkbox" bind:checked={settings.mixedImageOrderFirst} onchange={onSave} />
          <span class="text-xs font-bold text-slate-400">Sort images before words</span>
        </label>
      {/if}
    </div>

    <!-- Advanced Settings -->
    <div class="space-y-3">
      <label class="flex items-center gap-3 rounded-2xl border border-slate-700 bg-slate-950/40 px-4 py-3 hover:bg-slate-950/60 transition cursor-pointer">
        <input type="checkbox" bind:checked={settings.enforceClueGuessLimit} onchange={onSave} />
        <span class="text-sm font-medium text-slate-200">Require clue before guessing</span>
      </label>
      <label class="flex items-center gap-3 rounded-2xl border border-slate-700 bg-slate-950/40 px-4 py-3 hover:bg-slate-950/60 transition cursor-pointer">
        <input type="checkbox" bind:checked={settings.allowInfinityClue} onchange={onSave} />
        <span class="text-sm font-medium text-slate-200">Allow infinity clues (∞)</span>
      </label>
      <label class="flex items-center gap-3 rounded-2xl border border-slate-700 bg-slate-950/40 px-4 py-3 hover:bg-slate-950/60 transition cursor-pointer">
        <input type="checkbox" bind:checked={settings.observerChatEnabled} onchange={onSave} />
        <span class="text-sm font-medium text-slate-200">Observers can chat</span>
      </label>
      <label class="flex items-center gap-3 rounded-2xl border border-slate-700 bg-slate-950/40 px-4 py-3 hover:bg-slate-950/60 transition cursor-pointer">
        <input type="checkbox" bind:checked={settings.randomizeTeams} onchange={onSave} />
        <span class="text-sm font-medium text-slate-200">Auto-balance new players</span>
      </label>
    </div>

    <!-- Custom Colors -->
    <div class="grid gap-4 sm:grid-cols-2">
      <label class="block">
        <span class="text-xs font-bold text-slate-400">Team name</span>
        <input class="mt-2 w-full rounded-2xl border border-slate-700 bg-slate-950 px-4 py-3 text-slate-50" maxlength="30" bind:value={settings.teamNameBlue} onchange={onSave} placeholder="Libertarians" />
      </label>
      <label class="block">
        <span class="text-xs font-bold text-slate-400">Team name</span>
        <input class="mt-2 w-full rounded-2xl border border-slate-700 bg-slate-950 px-4 py-3 text-slate-50" maxlength="30" bind:value={settings.teamNameRed} onchange={onSave} placeholder="Monarchists" />
      </label>
      {#each [
        { team: 'blue' as const, fallback: '#3b82f6' },
        { team: 'red' as const, fallback: '#ef4444' }
      ] as picker (picker.team)}
        {@const currentColor = teamColor(picker.team, settings)}
        {@const teamName = displayTeamName(picker.team, settings)}
        <div class="relative block">
          <span class="text-xs font-bold text-slate-400">{teamName} color</span>
          <div class="mt-2 flex items-stretch gap-2 rounded-2xl border bg-slate-950/60 p-2 shadow-inner shadow-slate-950/30" style={`border-color: ${currentColor}66;`}>
            <button
              class="group flex min-w-0 flex-1 items-center gap-3 rounded-xl border border-slate-100/15 bg-slate-900/90 px-3 py-2 text-left shadow-lg shadow-slate-950/30 transition hover:-translate-y-0.5 hover:border-slate-100/30 hover:bg-slate-800/90"
              type="button"
              aria-label={colorPickerCtaLabel(teamName, currentColor)}
              aria-expanded={openColorPicker === picker.team}
              onclick={() => { openColorPicker = openColorPicker === picker.team ? null : picker.team; }}
            >
              <span class="h-8 w-8 shrink-0 rounded-lg border border-white/20 shadow-inner shadow-white/20" style={`background-color: ${currentColor};`}></span>
              <span class="min-w-0">
                <span class="block text-xs font-black uppercase tracking-[0.16em] text-slate-100">Choose</span>
                <span class="block truncate text-[11px] font-bold uppercase tracking-wider text-slate-500">{currentColor}</span>
              </span>
            </button>
            <input class="min-w-0 flex-1 rounded-xl border border-slate-800 bg-slate-900 px-3 text-xs font-black uppercase tracking-wider text-slate-300" value={colorInputValue(picker.team)} placeholder={picker.fallback} onchange={(event) => setColorInputValue(picker.team, event.currentTarget.value)} />
            <button class="shrink-0 rounded-xl border border-slate-700 bg-slate-900 px-3 py-2 text-xs font-black uppercase tracking-wider text-slate-300 transition hover:border-slate-300/70 hover:text-slate-100" type="button" onclick={() => resetTeamColor(picker.team)}>Reset</button>
          </div>

          {#if openColorPicker === picker.team}
            <button class="fixed inset-0 z-40 cursor-default bg-slate-950/20 backdrop-blur-[1px]" type="button" aria-label="Close color picker" onclick={closeColorPicker}></button>
            <div class="fixed left-1/2 top-1/2 z-50 w-[min(92vw,28rem)] -translate-x-1/2 -translate-y-1/2 overflow-hidden rounded-[1.75rem] border border-white/10 bg-slate-950/95 p-4 shadow-2xl shadow-slate-950/80" role="dialog" aria-modal="true" aria-label={`${teamName} color picker`}>
              <div class="pointer-events-none absolute -right-16 -top-20 h-44 w-44 rounded-full opacity-40 blur-3xl" style={`background-color: ${currentColor};`}></div>
              <div class="pointer-events-none absolute -bottom-16 -left-14 h-36 w-36 rounded-full bg-cyan-400/20 blur-3xl"></div>
              <div class="relative flex items-start justify-between gap-4">
                <div class="min-w-0">
                  <p class="text-sm font-black tracking-tight text-slate-50">Choose {teamName} color</p>
                  <p class="mt-1 text-xs font-bold uppercase tracking-[0.18em] text-slate-500">Preset grid or advanced picker</p>
                </div>
                <button class="rounded-full border border-white/10 bg-white/5 px-3 py-1.5 text-xs font-black uppercase tracking-wider text-slate-300 transition hover:bg-white/10 hover:text-white" type="button" onclick={closeColorPicker}>Close</button>
              </div>

              <div class="relative mt-4 flex items-center gap-3 rounded-2xl border border-white/10 bg-white/[0.03] p-3">
                <div class="h-12 w-12 shrink-0 rounded-2xl border border-white/20 shadow-inner shadow-white/20" style={`background-color: ${currentColor};`}></div>
                <div class="min-w-0 flex-1">
                  <p class="truncate text-lg font-black uppercase tracking-wide text-white">{currentColor}</p>
                  <p class="text-xs text-slate-500">Current team accent</p>
                </div>
                <label class="relative grid h-11 cursor-pointer place-items-center overflow-hidden rounded-xl border border-white/10 px-4 text-xs font-black uppercase tracking-[0.16em] text-white shadow-lg transition hover:scale-[1.02]" style={`background: linear-gradient(135deg, ${currentColor}, #0f172a);`}>
                  Advanced
                  <input class="absolute inset-0 h-full w-full cursor-pointer opacity-0" aria-label={colorInputLabel(picker.team, colorInputValue(picker.team), picker.fallback)} type="color" value={currentColor} onchange={(event) => setTeamColor(picker.team, event.currentTarget.value)} />
                </label>
              </div>

              <div class="relative mt-4 grid grid-cols-9 gap-1.5 rounded-2xl border border-white/10 bg-slate-900/70 p-2">
                {#each colorPresets as color (`${picker.team}-${color}`)}
                  <button
                    class={[
                      'h-8 rounded-lg border transition hover:scale-110 focus:outline-none focus:ring-2 focus:ring-white/70',
                      color.toLowerCase() === currentColor.toLowerCase() ? 'border-white shadow-lg shadow-white/20' : 'border-slate-100/15'
                    ]}
                    type="button"
                    title={color}
                    aria-label={`Set ${teamName} color to ${color}`}
                    style={`background-color: ${color}`}
                    onclick={() => setTeamColor(picker.team, color)}
                  ></button>
                {/each}
              </div>
            </div>
          {/if}
        </div>
      {/each}
    </div>

    {#if phase !== 'lobby'}
      <!-- Round Tools -->
      <div class="space-y-3 border-t border-slate-700/50 pt-4">
        <span class="text-xs font-black uppercase tracking-widest text-slate-500">Round tools</span>
        <div class="grid gap-3 sm:grid-cols-2">
          <button class="rounded-xl border border-amber-500/50 px-4 py-3 text-sm font-black text-amber-200 transition hover:bg-amber-500/10" onclick={onShuffleRoles}>Shuffle Card Roles</button>
          <button class="rounded-xl border border-amber-500/50 px-4 py-3 text-sm font-black text-amber-200 transition hover:bg-amber-500/10" onclick={onResetClue}>Reset Current Clue</button>
        </div>
        <button class="w-full rounded-xl border border-red-500/50 px-4 py-3 text-sm font-black text-red-200 transition hover:bg-red-500/10" onclick={onRestartMatch}>Restart Match (Back to Lobby)</button>
      </div>
    {/if}
  </fieldset>
  {/if}
</section>
