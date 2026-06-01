<script lang="ts">
  import { onMount } from 'svelte';

  import Sun from 'lucide-svelte/icons/sun';
  import Moon from 'lucide-svelte/icons/moon';
  import Terminal from 'lucide-svelte/icons/terminal';
  import SunMoon from 'lucide-svelte/icons/sun-moon';

  import { pressableButtonClasses } from './gameplay';
  import {
    applyTheme,
    darkModeThemes,
    lightModeThemes,
    prefersDarkScheme,
    readThemePreferences,
    resolveTheme,
    THEMES,
    themeMode,
    watchColorScheme,
    writeThemePreferences,
    type ThemeId,
    type ThemePreferences,
  } from './theme';

  type Props = {
    class?: string;
    // Output: the theme currently applied (mod override wins over saved prefs).
    effectiveThemeId?: ThemeId;
    // Saved theme preferences, shared so an external panel can reflect/edit them.
    themePreferences?: ThemePreferences;
    // A moderator-pushed theme applied for this session only (never persisted).
    sessionOverride?: ThemeId | null;
  };

  let {
    class: className = '',
    effectiveThemeId = $bindable('dark'),
    themePreferences = $bindable(readThemePreferences(localStorage)),
    sessionOverride = $bindable<ThemeId | null>(null),
  }: Props = $props();

  let systemPrefersDark = $state(prefersDarkScheme());
  let menuOpen = $state(false);

  $effect(() => {
    effectiveThemeId = sessionOverride ?? resolveTheme(themePreferences, systemPrefersDark);
  });

  $effect(() => {
    applyTheme(effectiveThemeId);
  });

  let ThemeIcon = $derived(
    themePreferences.auto
      ? SunMoon
      : effectiveThemeId === 'matrix'
        ? Terminal
        : themeMode(effectiveThemeId) === 'light'
          ? Sun
          : Moon,
  );
  let effectiveThemeLabel = $derived(THEMES.find((t) => t.id === effectiveThemeId)?.label ?? 'Dark');
  let buttonTitle = $derived(`Theme: ${effectiveThemeLabel}${themePreferences.auto ? ' (auto)' : ''} — click to change`);

  onMount(() => watchColorScheme((prefersDark) => (systemPrefersDark = prefersDark)));

  function updateThemePreferences(next: Partial<ThemePreferences>) {
    themePreferences = { ...themePreferences, ...next };
    writeThemePreferences(localStorage, themePreferences);
    // The user is taking control of their own theme; drop any moderator override.
    sessionOverride = null;
  }
</script>

<svelte:window onkeydown={(event) => { if (event.key === 'Escape' && menuOpen) menuOpen = false; }} />

<div class={['relative', className].filter(Boolean).join(' ')}>
  <button
    class={pressableButtonClasses('grid h-9 w-9 place-items-center rounded-full border border-slate-700 bg-slate-950/70 text-slate-100 hover:border-emerald-300/60 hover:text-emerald-100')}
    type="button"
    onclick={() => (menuOpen = !menuOpen)}
    title={buttonTitle}
    aria-label={buttonTitle}
    aria-haspopup="menu"
    aria-expanded={menuOpen}
  >
    <ThemeIcon class="h-4 w-4" />
  </button>
  {#if menuOpen}
    <!-- Click-away backdrop -->
    <button type="button" class="fixed inset-0 z-40 cursor-default" aria-label="Close theme menu" onclick={() => (menuOpen = false)}></button>
    <div class="absolute right-0 z-50 mt-2 w-60 rounded-2xl border border-slate-700 bg-slate-900 p-3 text-left shadow-2xl shadow-slate-950/50" role="menu">
      <p class="px-1 pb-2 text-xs font-black uppercase tracking-[0.16em] text-slate-400">Theme</p>
      <label class="flex items-center gap-2 rounded-xl border border-slate-800 bg-slate-950/60 px-3 py-2 text-xs text-slate-300 cursor-pointer">
        <input type="checkbox" checked={themePreferences.auto} onchange={(event) => updateThemePreferences({ auto: event.currentTarget.checked })} />
        Match system dark mode
      </label>
      {#if themePreferences.auto}
        <label class="mt-2 block text-xs text-slate-400">
          Dark mode theme
          <select class="mt-1 w-full rounded-xl border border-slate-700 bg-slate-950 px-3 py-2 text-sm text-slate-50" value={themePreferences.darkTheme} onchange={(event) => updateThemePreferences({ darkTheme: event.currentTarget.value as ThemeId })}>
            {#each darkModeThemes as theme (theme.id)}
              <option value={theme.id}>{theme.label}</option>
            {/each}
          </select>
        </label>
        <label class="mt-2 block text-xs text-slate-400">
          Light mode theme
          <select class="mt-1 w-full rounded-xl border border-slate-700 bg-slate-950 px-3 py-2 text-sm text-slate-50" value={themePreferences.lightTheme} onchange={(event) => updateThemePreferences({ lightTheme: event.currentTarget.value as ThemeId })}>
            {#each lightModeThemes as theme (theme.id)}
              <option value={theme.id}>{theme.label}</option>
            {/each}
          </select>
        </label>
      {:else}
        <div class="mt-2 grid gap-1.5" role="group" aria-label="Theme">
          {#each THEMES as theme (theme.id)}
            <button
              type="button"
              role="menuitemradio"
              aria-checked={effectiveThemeId === theme.id}
              class={['flex items-center justify-between rounded-xl border px-3 py-2 text-sm font-bold transition', effectiveThemeId === theme.id ? 'border-emerald-300/60 bg-emerald-300/10 text-emerald-100' : 'border-slate-700 bg-slate-950/60 text-slate-200 hover:border-emerald-300/40 hover:text-emerald-100'].join(' ')}
              onclick={() => updateThemePreferences({ manual: theme.id })}
            >
              {theme.label}
              {#if effectiveThemeId === theme.id}<span class="text-emerald-300">●</span>{/if}
            </button>
          {/each}
        </div>
      {/if}
    </div>
  {/if}
</div>
