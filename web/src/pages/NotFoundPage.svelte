<script lang="ts">
  import { appMeta } from '../lib/appMeta';
</script>

<main class="grid min-h-screen place-items-center bg-[oklch(14%_0.018_255)] px-6 text-slate-100">
  <section class="max-w-xl text-center">
    <p class="text-sm font-semibold uppercase tracking-[0.22em] text-emerald-300">{appMeta.name}</p>
    <h1 class="mt-4 text-4xl font-black tracking-tight">Room not found here.</h1>
    <a class="mt-8 inline-flex rounded-2xl bg-emerald-300 px-5 py-3 font-black text-slate-950" href="/">Return home</a>
  </section>
</main>
