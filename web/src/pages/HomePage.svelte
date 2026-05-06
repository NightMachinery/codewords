<script lang="ts">
  import { onMount } from 'svelte';

  import AuroraBackground from '../lib/backgrounds/AuroraBackground.svelte';
  import { api, defaultSettings } from '../lib/api';
  import { getOrCreateAuthToken } from '../lib/identity';
  import { roomPath } from '../lib/routes';

  let authToken = '';
  let displayName = $state('');
  let userId = $state('');
  let nameDraft = $state('');
  let roomDraft = $state('');
  let loading = $state(true);
  let creating = $state(false);
  let error = $state('');

  onMount(() => {
    authToken = getOrCreateAuthToken(localStorage);
    api
      .bootstrap(authToken)
      .then((identity) => {
        displayName = identity.displayName;
        userId = identity.userId;
        nameDraft = identity.displayName;
      })
      .catch((err: Error) => {
        error = err.message;
      })
      .finally(() => {
        loading = false;
      });
  });

  async function saveName() {
    const name = nameDraft.trim();
    if (!name) {
      error = 'Choose a display name before creating or joining a room.';
      return false;
    }
    const saved = await api.saveDisplayName(authToken, name);
    displayName = saved.displayName;
    nameDraft = saved.displayName;
    return true;
  }

  async function createRoom() {
    error = '';
    creating = true;
    try {
      if (!displayName || displayName !== nameDraft.trim()) {
        if (!(await saveName())) return;
      }
      const savedSettings = readCreatorSettings();
      const created = await api.createRoom(authToken, { ...defaultSettings, ...savedSettings, seed: Date.now() });
      window.location.href = roomPath(created.room.id);
    } catch (err) {
      error = err instanceof Error ? err.message : 'Could not create the room.';
    } finally {
      creating = false;
    }
  }

  async function joinRoom() {
    error = '';
    try {
      if (!displayName || displayName !== nameDraft.trim()) {
        if (!(await saveName())) return;
      }
      const id = parseRoomInput(roomDraft);
      if (!id) {
        error = 'Paste a room link or enter a room id.';
        return;
      }
      window.location.href = roomPath(id);
    } catch (err) {
      error = err instanceof Error ? err.message : 'Could not join the room.';
    }
  }

  function parseRoomInput(value: string) {
    const text = value.trim();
    if (!text) return '';
    try {
      const url = new URL(text);
      const match = /^\/rooms?\/([^/]+)\/?$/.exec(url.pathname);
      return match ? decodeURIComponent(match[1]) : text;
    } catch {
      return text.replace(/^#?\/?rooms?\//, '').trim();
    }
  }

  function readCreatorSettings() {
    if (!userId) return {};
    try {
      const parsed = JSON.parse(localStorage.getItem(`codewords.creatorSettings.${userId}`) ?? '{}');
      delete parsed.seed;
      return parsed;
    } catch {
      return {};
    }
  }
</script>

<main class="relative min-h-screen w-full overflow-hidden bg-[oklch(10%_0.025_255)] text-slate-100">
  <AuroraBackground intensity={1.0} speed={1.0} />

  <div class="hero-shell relative z-10 flex min-h-screen flex-col px-5 py-7 sm:px-8 lg:px-12">
    <nav class="flex items-center justify-center sm:justify-start">
      <a class="rounded-full border border-emerald-200/20 bg-slate-950/35 px-5 py-2 text-sm font-black uppercase tracking-[0.32em] text-emerald-100 shadow-2xl shadow-cyan-950/30 outline-none backdrop-blur-md transition hover:border-emerald-100/45 focus-visible:ring-2 focus-visible:ring-emerald-200" href="/">
        CODEWORDS
      </a>
    </nav>

    <section class="grid flex-1 place-items-center py-12">
      <section class="w-full max-w-[25rem] rounded-[2rem] border border-emerald-100/15 bg-[oklch(16%_0.026_255_/_0.78)] p-5 shadow-[0_24px_90px_oklch(5%_0.03_255_/_0.62)] backdrop-blur-xl sm:p-6" aria-label="Room entry">
        {#if loading}
          <p class="py-10 text-center text-sm font-bold text-slate-300">Loading identity...</p>
        {:else}
          <div class="space-y-5">
            <label class="block">
              <span class="text-xs font-black uppercase tracking-[0.2em] text-emerald-100/80">Display name</span>
              <input
                class="mt-2 w-full rounded-2xl border border-slate-500/35 bg-[oklch(11%_0.025_255_/_0.86)] px-4 py-3 text-slate-50 outline-none ring-emerald-200 transition placeholder:text-slate-500 focus:border-emerald-100/60 focus:ring-2"
                bind:value={nameDraft}
                maxlength="40"
                placeholder="Your table name"
              />
            </label>

            <button
              class="w-full rounded-2xl bg-emerald-200 px-5 py-3 font-black text-[oklch(14%_0.025_255)] shadow-lg shadow-emerald-950/30 transition hover:bg-emerald-100 disabled:cursor-not-allowed disabled:opacity-60"
              disabled={creating}
              onclick={createRoom}
            >
              {creating ? 'Creating...' : 'Create room'}
            </button>

            <div class="grid grid-cols-[1fr_auto] gap-2">
              <input
                class="min-w-0 rounded-2xl border border-slate-500/35 bg-[oklch(11%_0.025_255_/_0.86)] px-4 py-3 text-slate-50 outline-none ring-cyan-200 transition placeholder:text-slate-500 focus:border-cyan-100/60 focus:ring-2"
                bind:value={roomDraft}
                placeholder="Paste room link or id"
              />
              <button
                class="rounded-2xl border border-cyan-100/35 bg-cyan-100/5 px-5 py-3 font-black text-cyan-50 transition hover:border-cyan-100/70 hover:bg-cyan-100/10 disabled:opacity-60"
                onclick={joinRoom}
              >
                Join
              </button>
            </div>
          </div>
        {/if}

        {#if error}
          <p class="mt-5 rounded-2xl border border-red-300/45 bg-red-400/15 px-4 py-3 text-sm font-bold text-red-50">{error}</p>
        {/if}
      </section>
    </section>
  </div>
</main>
