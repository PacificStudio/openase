<script lang="ts">
  import type { ProjectConversationCommandOutputEntry } from './project-conversation-transcript-state'

  let { entry }: { entry: ProjectConversationCommandOutputEntry } = $props()
</script>

<div
  class="space-y-2 rounded-2xl border border-slate-700/20 bg-slate-950 px-3 py-3 text-sm text-slate-100"
>
  <div class="text-[10px] font-semibold tracking-[0.16em] text-slate-400 uppercase">runtime</div>
  <div class="font-medium">Command output</div>
  <div class="text-xs text-slate-400">
    <span>{entry.stream}</span>
    {#if entry.phase}
      <span> · {entry.phase}</span>
    {/if}
    <span> · {entry.snapshot ? 'snapshot' : 'delta'}</span>
  </div>
  <pre
    class="overflow-x-auto rounded-xl bg-black/30 px-3 py-2 text-xs leading-5 whitespace-pre-wrap">{entry.content}</pre>
</div>
