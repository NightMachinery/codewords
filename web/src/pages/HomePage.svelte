<script lang="ts">
  import { onMount } from 'svelte';

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
  <div class="aurora-background" aria-hidden="true">
    <div class="aurora-ribbon aurora-ribbon-one"></div>
    <div class="aurora-ribbon aurora-ribbon-two"></div>
    <div class="aurora-ribbon aurora-ribbon-three"></div>
    <div class="aurora-ribbon aurora-ribbon-four"></div>
  </div>

  <div class="relative z-10 flex min-h-screen flex-col px-5 py-7 sm:px-8 lg:px-12">
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

<style>
  .aurora-background {
    position: fixed;
    inset: 0;
    overflow: hidden;
    pointer-events: none;
    background:
      radial-gradient(circle at 50% 118%, oklch(21% 0.035 258 / 0.86), transparent 46%),
      linear-gradient(180deg, oklch(8% 0.03 260), oklch(13% 0.035 242) 58%, oklch(7% 0.025 265));
  }

  .aurora-background::before {
    content: '';
    position: absolute;
    inset: -18%;
    background:
      radial-gradient(circle at 18% 18%, oklch(71% 0.16 165 / 0.18), transparent 24%),
      radial-gradient(circle at 82% 22%, oklch(70% 0.13 225 / 0.16), transparent 28%),
      radial-gradient(circle at 50% 4%, oklch(67% 0.15 305 / 0.12), transparent 26%);
    filter: blur(28px);
  }

  .aurora-background::after {
    content: '';
    position: absolute;
    inset: 0;
    background: linear-gradient(180deg, transparent, oklch(6% 0.024 260 / 0.54) 76%);
  }

  .aurora-ribbon {
    position: absolute;
    left: 50%;
    top: 8%;
    width: 120vw;
    height: 34vh;
    border-radius: 9999px;
    filter: blur(34px);
    mix-blend-mode: screen;
    --x0: -58%;
    --y0: -12%;
    --r0: -14deg;
    --sx0: 0.96;
    --sy0: 0.82;
    --x1: -48%;
    --y1: 4%;
    --r1: 7deg;
    --sx1: 1.12;
    --sy1: 1.04;
    --x2: -44%;
    --y2: -6%;
    --r2: 16deg;
    --sx2: 1.02;
    --sy2: 0.94;
    opacity: 0.72;
    transform: translate3d(var(--x1), var(--y1), 0) rotate(var(--r1)) scale(var(--sx1), var(--sy1));
    transform-origin: 50% 50%;
    will-change: transform, opacity;
    animation: aurora-drift 17s cubic-bezier(0.22, 1, 0.36, 1) infinite alternate;
  }

  .aurora-ribbon-one {
    --x0: -56%;
    --y0: -8%;
    --r0: -13deg;
    --sx0: 0.98;
    --sy0: 0.84;
    --x1: -48%;
    --y1: 3%;
    --r1: -5deg;
    --sx1: 1.1;
    --sy1: 1.02;
    --x2: -42%;
    --y2: -4%;
    --r2: 5deg;
    --sx2: 1.02;
    --sy2: 0.94;
    background: conic-gradient(from 248deg, transparent, oklch(76% 0.19 160 / 0.78), oklch(70% 0.14 205 / 0.58), transparent 64%);
  }

  .aurora-ribbon-two {
    top: 22%;
    height: 28vh;
    background: conic-gradient(from 102deg, transparent, oklch(70% 0.15 225 / 0.6), oklch(66% 0.17 292 / 0.42), transparent 58%);
    opacity: 0.64;
    --x0: -46%;
    --y0: -15%;
    --r0: 2deg;
    --sx0: 1.04;
    --sy0: 0.78;
    --x1: -53%;
    --y1: -2%;
    --r1: 10deg;
    --sx1: 1.16;
    --sy1: 0.98;
    --x2: -42%;
    --y2: 4%;
    --r2: 18deg;
    --sx2: 1.06;
    --sy2: 0.86;
    animation-duration: 23s;
    animation-delay: -8s;
  }

  .aurora-ribbon-three {
    top: 38%;
    height: 22vh;
    background: radial-gradient(ellipse at 50% 50%, oklch(78% 0.16 178 / 0.45), oklch(69% 0.16 250 / 0.26) 42%, transparent 72%);
    opacity: 0.52;
    --x0: -62%;
    --y0: -18%;
    --r0: -20deg;
    --sx0: 0.86;
    --sy0: 0.7;
    --x1: -54%;
    --y1: -7%;
    --r1: -12deg;
    --sx1: 1.02;
    --sy1: 0.9;
    --x2: -49%;
    --y2: -16%;
    --r2: -4deg;
    --sx2: 0.92;
    --sy2: 0.78;
    animation-duration: 29s;
    animation-delay: -14s;
  }

  .aurora-ribbon-four {
    top: -2%;
    height: 46vh;
    background: radial-gradient(ellipse at 50% 50%, oklch(64% 0.18 310 / 0.24), oklch(75% 0.17 155 / 0.22) 45%, transparent 74%);
    opacity: 0.46;
    --x0: -52%;
    --y0: -26%;
    --r0: 12deg;
    --sx0: 1.06;
    --sy0: 0.86;
    --x1: -45%;
    --y1: -18%;
    --r1: 21deg;
    --sx1: 1.2;
    --sy1: 1;
    --x2: -56%;
    --y2: -14%;
    --r2: 28deg;
    --sx2: 1.08;
    --sy2: 0.9;
    animation-duration: 31s;
    animation-delay: -19s;
  }

  @keyframes aurora-drift {
    0% {
      opacity: 0.44;
      transform: translate3d(var(--x0), var(--y0), 0) rotate(var(--r0)) scale(var(--sx0), var(--sy0));
    }
    52% {
      opacity: 0.82;
      transform: translate3d(var(--x1), var(--y1), 0) rotate(var(--r1)) scale(var(--sx1), var(--sy1));
    }
    100% {
      opacity: 0.58;
      transform: translate3d(var(--x2), var(--y2), 0) rotate(var(--r2)) scale(var(--sx2), var(--sy2));
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .aurora-ribbon {
      animation: none;
      opacity: 0.58;
    }
  }
</style>
