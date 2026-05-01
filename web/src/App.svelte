<script lang="ts">
  import { appMeta } from './lib/appMeta';
</script>

<main class="min-h-screen bg-slate-950 text-slate-100">
  <section class="mx-auto flex min-h-screen w-full max-w-5xl flex-col items-center justify-center gap-8 px-6 py-16 text-center">
    <p class="rounded-full border border-emerald-400/30 bg-emerald-400/10 px-4 py-2 text-sm font-semibold text-emerald-200">
      Greenfield scaffold ready
    </p>

    <div class="space-y-4">
      <h1 class="text-5xl font-black tracking-tight text-white sm:text-7xl">{appMeta.name}</h1>
      <p class="mx-auto max-w-2xl text-lg leading-8 text-slate-300 sm:text-xl">{appMeta.tagline}</p>
    </div>

    <div class="grid w-full gap-4 text-left sm:grid-cols-3">
      <article class="rounded-3xl border border-white/10 bg-white/5 p-6 shadow-2xl shadow-black/20">
        <h2 class="text-lg font-bold text-white">Local-first</h2>
        <p class="mt-2 text-sm leading-6 text-slate-300">Built for intranet play without external fonts, CDNs, analytics, or hosted services.</p>
      </article>
      <article class="rounded-3xl border border-white/10 bg-white/5 p-6 shadow-2xl shadow-black/20">
        <h2 class="text-lg font-bold text-white">Realtime-ready</h2>
        <p class="mt-2 text-sm leading-6 text-slate-300">The Go backend starts with a health endpoint and room APIs will grow from this scaffold.</p>
      </article>
      <article class="rounded-3xl border border-white/10 bg-white/5 p-6 shadow-2xl shadow-black/20">
        <h2 class="text-lg font-bold text-white">Self-hosted</h2>
        <p class="mt-2 text-sm leading-6 text-slate-300">Caddy and tmux lifecycle scripts are prepared for later production hardening.</p>
      </article>
    </div>
  </section>
</main>
