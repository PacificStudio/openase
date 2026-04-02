<script lang="ts">
  import type { ProjectConversationToolCallEntry } from './project-conversation-transcript-state'

  let { entry }: { entry: ProjectConversationToolCallEntry } = $props()

  function asRecord(value: unknown) {
    return value && typeof value === 'object' && !Array.isArray(value)
      ? (value as Record<string, unknown>)
      : null
  }

  function readString(...keys: string[]) {
    const record = asRecord(entry.arguments)
    for (const key of keys) {
      const value = record?.[key]
      if (typeof value === 'string' && value.trim()) {
        return value
      }
    }
    return ''
  }

  function toolTitle() {
    switch (entry.tool) {
      case 'functions.exec_command':
        return 'Run command'
      case 'functions.apply_patch':
        return 'Apply patch'
      case 'functions.write_stdin':
        return 'Write stdin'
      default:
        return 'Tool call'
    }
  }

  const command = $derived(readString('cmd', 'command'))
  const target = $derived(readString('path', 'file', 'target', 'workdir'))
  const patch = $derived(readString('patch'))

  function formatJSON(value: unknown) {
    const formatted = JSON.stringify(value ?? {}, null, 2)
    return formatted ?? '{}'
  }
</script>

<div class="space-y-2 rounded-2xl border border-violet-500/20 bg-violet-500/5 p-3 text-sm">
  <div class="text-[10px] font-semibold tracking-[0.16em] text-violet-700 uppercase">runtime</div>
  <div class="font-medium text-violet-950">{toolTitle()}</div>
  <div class="text-xs text-violet-900">
    <span class="font-mono">{entry.tool}</span>
  </div>

  {#if command}
    <div class="rounded-xl border border-violet-500/20 bg-white/80 px-3 py-2">
      <div class="mb-1 text-[10px] font-semibold tracking-[0.14em] text-violet-700 uppercase">
        command
      </div>
      <pre class="font-mono text-xs leading-5 whitespace-pre-wrap text-violet-950">{command}</pre>
    </div>
  {/if}

  {#if target}
    <div class="rounded-xl border border-violet-500/20 bg-white/80 px-3 py-2">
      <div class="mb-1 text-[10px] font-semibold tracking-[0.14em] text-violet-700 uppercase">
        target
      </div>
      <div class="font-mono text-xs leading-5 text-violet-950">{target}</div>
    </div>
  {/if}

  {#if patch}
    <details class="rounded-xl border border-violet-500/20 bg-white/80">
      <summary class="cursor-pointer px-3 py-2 text-xs font-medium text-violet-900">
        Patch preview
      </summary>
      <pre
        class="overflow-x-auto border-t border-violet-500/20 px-3 py-2 text-xs leading-5 whitespace-pre-wrap text-violet-950">{patch}</pre>
    </details>
  {/if}

  <details class="rounded-xl border border-violet-500/20 bg-white/80">
    <summary class="cursor-pointer px-3 py-2 text-xs font-medium text-violet-900">Arguments</summary
    >
    <pre
      class="overflow-x-auto border-t border-violet-500/20 px-3 py-2 text-xs leading-5 whitespace-pre-wrap text-violet-950">{formatJSON(
        entry.arguments,
      )}</pre>
  </details>
</div>
