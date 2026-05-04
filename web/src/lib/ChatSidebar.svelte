<script lang="ts">
  import type { ChatMessage } from './api';

  interface Props {
    messages: ChatMessage[];
    draft: string;
    canChat: boolean;
    onSend: (body: string) => void;
  }

  let { messages, draft = $bindable(), canChat, onSend }: Props = $props();
  let expanded = $state(false);

  function handleSubmit() {
    if (draft.trim()) {
      onSend(draft);
      draft = '';
    }
  }
</script>

<aside class={['fixed top-0 right-0 z-40 h-full border-l border-slate-700/70 bg-slate-900/95 shadow-2xl transition-all duration-300 flex flex-col', expanded ? 'w-80' : 'w-12']}>
  <button 
    class="flex h-12 w-full items-center justify-center border-b border-slate-700/50 hover:bg-slate-800 transition"
    onclick={() => expanded = !expanded}
    title={expanded ? 'Collapse chat' : 'Expand chat'}
  >
    {#if expanded}
      <span class="text-xs font-black uppercase tracking-widest text-slate-400">Chat</span>
    {:else}
      <span class="text-xl">💬</span>
    {/if}
  </button>

  {#if expanded}
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
          class="rounded-xl bg-emerald-300 px-3 py-2 font-black text-slate-950 transition hover:bg-emerald-200 disabled:opacity-50" 
          disabled={!canChat || !draft.trim()} 
          onclick={handleSubmit}
        >
          Send
        </button>
      </div>
    </div>
  {/if}
</aside>
