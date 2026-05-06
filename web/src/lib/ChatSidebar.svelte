<script lang="ts">
  import { onMount } from 'svelte';
  import type { ChatMessage } from './api';
  import { chatToggleEventName, pressableButtonClasses } from './gameplay';

  interface Props {
    messages: ChatMessage[];
    draft: string;
    canChat: boolean;
    onSend: (body: string) => void;
  }

  let { messages, draft = $bindable(), canChat, onSend }: Props = $props();
  let expanded = $state(false);

  onMount(() => {
    const toggle = () => (expanded = !expanded);
    window.addEventListener(chatToggleEventName, toggle);
    return () => window.removeEventListener(chatToggleEventName, toggle);
  });

  function handleSubmit() {
    if (draft.trim()) {
      onSend(draft);
      draft = '';
    }
  }
</script>

{#if !expanded}
  <button
    class={pressableButtonClasses('fixed right-3 top-16 z-40 grid h-11 w-11 place-items-center rounded-full border border-slate-700/70 bg-slate-900/95 text-emerald-100 shadow-2xl hover:border-emerald-300/60 hover:bg-slate-800')}
    onclick={() => expanded = true}
    title="Expand chat"
    aria-label="Expand chat"
  >
    <svg class="h-5 w-5" viewBox="0 0 24 24" aria-hidden="true">
      <path fill="currentColor" d="M4 5.5A3.5 3.5 0 0 1 7.5 2h9A3.5 3.5 0 0 1 20 5.5v6A3.5 3.5 0 0 1 16.5 15H10l-5.2 4.6A.8.8 0 0 1 3.5 19v-4.4A3.5 3.5 0 0 1 2 11.7V5.5Z" opacity="0.35" />
      <path fill="none" stroke="currentColor" stroke-linecap="round" stroke-width="1.8" d="M8 8h8M8 11h5" />
    </svg>
  </button>
{:else}
<aside class="fixed bottom-24 right-0 top-0 z-40 flex w-80 flex-col border-b border-l border-slate-700/70 bg-slate-900/95 shadow-2xl transition-all duration-300">
  <button
    class={pressableButtonClasses('flex h-12 w-full items-center justify-center border-b border-slate-700/50 hover:bg-slate-800')}
    onclick={() => expanded = !expanded}
    title={expanded ? 'Collapse chat' : 'Expand chat'}
  >
    <span class="text-xs font-black uppercase tracking-widest text-slate-400">Chat</span>
  </button>

    <div class="flex-1 overflow-y-auto p-4 space-y-3">
      {#each messages as message (message.id)}
        <article class="rounded-2xl border border-slate-700 bg-slate-950/80 px-4 py-3">
          <p class="text-[10px] font-black uppercase tracking-wider text-emerald-300">{message.displayName || 'Player'}</p>
          <p class="mt-1 break-words text-sm leading-relaxed text-slate-100">{message.body}</p>
        </article>
      {:else}
        <p class="text-center text-sm text-slate-500 mt-10">No messages yet.</p>
      {/each}
    </div>

    <div class="p-4 border-t border-slate-700/50 bg-slate-950/50">
      <div class="flex gap-2">
        <input
          class="flex-1 min-w-0 rounded-xl border border-slate-700 bg-slate-950 px-3 py-2 text-sm text-slate-50 outline-none ring-emerald-300 transition focus:ring-2 disabled:opacity-50"
          bind:value={draft}
          maxlength="1000"
          placeholder={canChat ? 'Type a message...' : 'Chat disabled'}
          disabled={!canChat}
          onkeydown={(e) => e.key === 'Enter' && handleSubmit()}
        />
        <button
          class={pressableButtonClasses('rounded-xl bg-emerald-300 px-3 py-2 font-black text-slate-950 hover:bg-emerald-200 disabled:opacity-50')}
          disabled={!canChat || !draft.trim()}
          onclick={handleSubmit}
        >
          Send
        </button>
      </div>
    </div>
</aside>
{/if}
