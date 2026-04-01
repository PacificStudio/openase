<script lang="ts">
  import type { ProjectConversationToolCallEntry } from './project-conversation-transcript-state'

  let { entry }: { entry: ProjectConversationToolCallEntry } = $props()

  function formatJSON(value: unknown) {
    const formatted = JSON.stringify(value ?? {}, null, 2)
    return formatted ?? '{}'
  }
</script>

<div class="space-y-2 rounded-2xl border border-violet-500/20 bg-violet-500/5 p-3 text-sm">
  <div class="text-[10px] font-semibold tracking-[0.16em] text-violet-700 uppercase">runtime</div>
  <div class="font-medium text-violet-950">Tool call</div>
  <div class="text-xs text-violet-900">
    <span class="font-mono">{entry.tool}</span>
  </div>

  <details class="rounded-xl border border-violet-500/20 bg-white/80">
    <summary class="cursor-pointer px-3 py-2 text-xs font-medium text-violet-900">Arguments</summary
    >
    <pre
      class="overflow-x-auto border-t border-violet-500/20 px-3 py-2 text-xs leading-5 whitespace-pre-wrap text-violet-950">{formatJSON(
        entry.arguments,
      )}</pre>
  </details>
</div>
