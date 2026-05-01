<script lang="ts">
  import { onMount } from 'svelte';

  import { api, defaultSettings } from '../lib/api';
  import { getOrCreateAuthToken } from '../lib/identity';
  import { roomPath } from '../lib/routes';

  let authToken = '';
  let displayName = $state('');
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
      const created = await api.createRoom(authToken, { ...defaultSettings, seed: Date.now() });
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
</script>

<main class="min-h-screen w-full overflow-x-hidden bg-[oklch(14%_0.018_255)] text-slate-100">
  <div class="mx-auto flex min-h-screen w-full max-w-6xl flex-col px-5 py-8 sm:px-8">
    <nav class="flex items-center justify-between rounded-full border border-slate-700/70 bg-slate-900/70 px-5 py-3 shadow-2xl shadow-slate-950/40">
      <a class="text-lg font-black tracking-tight text-slate-50" href="/">Codewords</a>
      <span class="text-sm text-slate-300">Self-hosted rooms, local assets, no accounts.</span>
    </nav>

    <section class="grid flex-1 items-center gap-12 py-20 lg:grid-cols-[1fr_25rem] lg:py-28">
      <div class="max-w-5xl">
        <p class="mb-5 max-w-xl text-sm font-semibold uppercase tracking-[0.22em] text-emerald-300">Private team wordplay</p>
        <h1 class="max-w-5xl text-5xl font-black leading-[0.95] tracking-[-0.05em] text-slate-50 sm:text-7xl lg:text-8xl">
          Start a Codewords table in seconds.
        </h1>
        <p class="mt-7 max-w-2xl text-lg leading-8 text-slate-300">
          Create a room, share the link, and keep every identity on your own server. Works on plain HTTP for local networks.
        </p>
      </div>

      <section class="rounded-[2rem] border border-slate-700/70 bg-slate-900/80 p-5 shadow-2xl shadow-slate-950/40">
        {#if loading}
          <p class="py-10 text-center text-slate-300">Loading identity...</p>
        {:else}
          <div class="space-y-5">
            <label class="block">
              <span class="text-sm font-semibold text-slate-200">Display name</span>
              <input
                class="mt-2 w-full rounded-2xl border border-slate-700 bg-slate-950 px-4 py-3 text-slate-50 outline-none ring-emerald-300 transition focus:ring-2"
                bind:value={nameDraft}
                maxlength="40"
                placeholder="Your table name"
              />
            </label>

            <button
              class="w-full rounded-2xl bg-emerald-300 px-5 py-3 font-black text-slate-950 transition hover:bg-emerald-200 disabled:cursor-not-allowed disabled:opacity-60"
              disabled={creating}
              onclick={createRoom}
            >
              {creating ? 'Creating...' : 'Create room'}
            </button>

            <div class="grid grid-cols-[1fr_auto] gap-2">
              <input
                class="rounded-2xl border border-slate-700 bg-slate-950 px-4 py-3 text-slate-50 outline-none ring-blue-300 transition focus:ring-2"
                bind:value={roomDraft}
                placeholder="Paste room link or id"
              />
              <button
                class="rounded-2xl border border-slate-600 px-5 py-3 font-bold text-slate-100 transition hover:border-blue-300 hover:text-blue-200"
                onclick={joinRoom}
              >
                Join
              </button>
            </div>
          </div>
        {/if}

        {#if error}
          <p class="mt-5 rounded-2xl border border-red-400/40 bg-red-400/10 px-4 py-3 text-sm text-red-100">{error}</p>
        {/if}
      </section>
    </section>
  </div>
</main>
