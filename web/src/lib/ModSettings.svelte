<script lang="ts">
  import type { PictureAsset, Settings, Wordpack } from './api';
  import { cardModeFromImageCount, imageCountForMode } from './gameplay';

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

  function setCardMode(mode: 'words' | 'images' | 'mixed') {
    settings.imageCardCount = imageCountForMode(mode, settings.imageCardCount);
    onSave();
  }

  function setMixedImageCount(count: string) {
    settings.imageCardCount = Number.parseInt(count, 10);
    onSave();
  }

  function colorInputLabel(team: 'Blue' | 'Red', color: string | undefined, fallback: string) {
    return `${team} team color ${color || fallback}`;
  }
</script>

<section class="rounded-[2rem] border border-slate-700/70 bg-slate-900/80 p-6 shadow-2xl shadow-slate-950/30">
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
        <span class="text-xs font-bold text-slate-400">Blue team color</span>
        <div class="mt-2 flex gap-2 rounded-2xl border bg-slate-950/60 p-2 shadow-inner shadow-blue-950/20" style={`border-color: ${settings.customColorBlue ? settings.customColorBlue : '#3b82f6'}66;`}>
          <span class="relative grid h-12 w-20 shrink-0 place-items-center overflow-hidden rounded-xl border border-slate-100/15 shadow-lg shadow-slate-950/30" style={`background-color: ${settings.customColorBlue || '#3b82f6'};`}>
            <span class="rounded-full bg-slate-950/70 px-2 py-1 text-[10px] font-black uppercase tracking-wider text-slate-50">Pick</span>
            <input class="absolute inset-0 h-full w-full cursor-pointer opacity-0" aria-label={colorInputLabel('Blue', settings.customColorBlue, '#3b82f6')} type="color" value={settings.customColorBlue || '#3b82f6'} onchange={(event) => { settings.customColorBlue = event.currentTarget.value; onSave(); }} />
          </span>
          <span class="flex min-w-0 flex-1 items-center rounded-xl border border-slate-800 bg-slate-900 px-3 text-xs font-black uppercase tracking-wider text-slate-300">{settings.customColorBlue || '#3b82f6'}</span>
          <button class="flex-1 rounded-xl border border-slate-700 bg-slate-900 px-3 py-2 text-xs font-black uppercase tracking-wider text-slate-300 transition hover:border-blue-300/70 hover:text-blue-100" type="button" onclick={() => { settings.customColorBlue = ''; onSave(); }}>Reset</button>
        </div>
      </label>
      <label class="block">
        <span class="text-xs font-bold text-slate-400">Red team color</span>
        <div class="mt-2 flex gap-2 rounded-2xl border bg-slate-950/60 p-2 shadow-inner shadow-red-950/20" style={`border-color: ${settings.customColorRed ? settings.customColorRed : '#ef4444'}66;`}>
          <span class="relative grid h-12 w-20 shrink-0 place-items-center overflow-hidden rounded-xl border border-slate-100/15 shadow-lg shadow-slate-950/30" style={`background-color: ${settings.customColorRed || '#ef4444'};`}>
            <span class="rounded-full bg-slate-950/70 px-2 py-1 text-[10px] font-black uppercase tracking-wider text-slate-50">Pick</span>
            <input class="absolute inset-0 h-full w-full cursor-pointer opacity-0" aria-label={colorInputLabel('Red', settings.customColorRed, '#ef4444')} type="color" value={settings.customColorRed || '#ef4444'} onchange={(event) => { settings.customColorRed = event.currentTarget.value; onSave(); }} />
          </span>
          <span class="flex min-w-0 flex-1 items-center rounded-xl border border-slate-800 bg-slate-900 px-3 text-xs font-black uppercase tracking-wider text-slate-300">{settings.customColorRed || '#ef4444'}</span>
          <button class="flex-1 rounded-xl border border-slate-700 bg-slate-900 px-3 py-2 text-xs font-black uppercase tracking-wider text-slate-300 transition hover:border-red-300/70 hover:text-red-100" type="button" onclick={() => { settings.customColorRed = ''; onSave(); }}>Reset</button>
        </div>
      </label>
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
