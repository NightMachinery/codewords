<script lang="ts">
  import Eye from 'lucide-svelte/icons/eye';
  import RotateCcw from 'lucide-svelte/icons/rotate-ccw';
  import SkipForward from 'lucide-svelte/icons/skip-forward';

  import type { ConfirmationRequest, ConfirmationTone } from './confirmation';

  interface Props {
    request: ConfirmationRequest | null;
    onConfirm: () => void;
    onCancel: () => void;
  }

  let { request, onConfirm, onCancel }: Props = $props();
  let confirmButton: HTMLButtonElement | null = $state(null);

  const toneClasses: Record<ConfirmationTone, { shell: string; icon: string; confirm: string; eyebrow: string }> = {
    reveal: {
      shell: 'border-emerald-200/25 bg-slate-950/95 shadow-emerald-950/40',
      icon: 'border-emerald-200/30 bg-emerald-300/15 text-emerald-100',
      confirm: 'bg-emerald-300 text-slate-950 hover:bg-emerald-200',
      eyebrow: 'text-emerald-200',
    },
    pass: {
      shell: 'border-cyan-200/25 bg-slate-950/95 shadow-cyan-950/40',
      icon: 'border-cyan-200/30 bg-cyan-300/15 text-cyan-100',
      confirm: 'bg-cyan-200 text-slate-950 hover:bg-cyan-100',
      eyebrow: 'text-cyan-100',
    },
    danger: {
      shell: 'border-red-300/35 bg-slate-950/95 shadow-red-950/40',
      icon: 'border-red-200/30 bg-red-400/15 text-red-100',
      confirm: 'bg-red-300 text-red-950 hover:bg-red-200',
      eyebrow: 'text-red-100',
    },
  };

  let tone = $derived(request ? toneClasses[request.tone] : toneClasses.reveal);

  $effect(() => {
    if (request && confirmButton) confirmButton.focus();
  });

  function handleKeydown(event: KeyboardEvent) {
    if (request && event.key === 'Escape') onCancel();
  }
</script>

<svelte:window onkeydown={handleKeydown} />

{#if request}
  <button
    class="fixed inset-0 z-40 cursor-default bg-slate-950/70 backdrop-blur-sm"
    type="button"
    aria-label="Cancel confirmation"
    onclick={onCancel}
  ></button>
  <div
    class={['fixed left-1/2 top-1/2 z-50 w-[min(92vw,26rem)] -translate-x-1/2 -translate-y-1/2 overflow-hidden rounded-[1.5rem] border p-5 text-slate-50 shadow-2xl', tone.shell].join(' ')}
    role="dialog"
    aria-modal="true"
    aria-labelledby="confirmation-title"
    aria-describedby="confirmation-message"
    tabindex="-1"
  >
    <div class="pointer-events-none absolute inset-x-5 top-0 h-px bg-slate-100/20"></div>
    <div class="relative flex gap-4">
      <div class={['grid h-12 w-12 shrink-0 place-items-center rounded-2xl border', tone.icon].join(' ')}>
        {#if request.kind === 'reveal'}
          <Eye class="h-5 w-5" />
        {:else if request.kind === 'pass'}
          <SkipForward class="h-5 w-5" />
        {:else}
          <RotateCcw class="h-5 w-5" />
        {/if}
      </div>
      <div class="min-w-0 flex-1">
        <p class={['text-xs font-black uppercase tracking-[0.2em]', tone.eyebrow].join(' ')}>{request.kind}</p>
        <h2 id="confirmation-title" class="mt-1 text-2xl font-black tracking-tight text-slate-50">{request.title}</h2>
        <p id="confirmation-message" class="mt-2 text-sm font-semibold leading-6 text-slate-300">{request.message}</p>
      </div>
    </div>

    {#if request.cardPreview}
      <figure class="relative mt-5 overflow-hidden rounded-2xl border border-slate-700 bg-slate-900/80 p-2">
        <div class="mx-auto aspect-[2/3] max-h-72 w-full max-w-48 overflow-hidden rounded-xl">
          <img
            class="h-full w-full object-cover"
            src={request.cardPreview.imageUrl}
            alt={request.cardPreview.label}
          />
        </div>
        <figcaption class="mt-2 text-center text-xs font-black uppercase tracking-[0.18em] text-slate-400">
          {request.cardPreview.label}
        </figcaption>
      </figure>
    {/if}

    <div class="relative mt-6 grid gap-2 min-[420px]:grid-cols-[1fr_auto]">
      <button
        class="rounded-2xl border border-slate-600 bg-slate-900/80 px-4 py-3 text-sm font-black text-slate-200 transition hover:border-slate-400 hover:text-slate-50 active:translate-y-px"
        type="button"
        onclick={onCancel}
      >
        {request.cancelLabel}
      </button>
      <button
        bind:this={confirmButton}
        class={['rounded-2xl px-5 py-3 text-sm font-black shadow-lg transition active:translate-y-px', tone.confirm].join(' ')}
        type="button"
        onclick={onConfirm}
      >
        {request.confirmLabel}
      </button>
    </div>
  </div>
{/if}
