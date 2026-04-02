<script lang="ts">
  import type { ProjectConversationTaskStatusEntry } from './project-conversation-transcript-state'

  let { entry }: { entry: ProjectConversationTaskStatusEntry } = $props()

  function readString(...keys: string[]) {
    const raw = entry.raw ?? {}
    for (const key of keys) {
      const value = raw[key]
      if (typeof value === 'string' && value.trim()) {
        return value
      }
    }
    return ''
  }

  function otherPayload() {
    const raw = entry.raw ?? {}
    const hidden = new Set([
      'status',
      'stream',
      'phase',
      'command',
      'file',
      'path',
      'target',
      'patch',
      'text',
      'message',
    ])
    const remaining = Object.fromEntries(
      Object.entries(raw).filter(([key, value]) => !hidden.has(key) && value != null),
    )
    return Object.keys(remaining).length > 0 ? remaining : null
  }

  function formatJSON(value: unknown) {
    const formatted = JSON.stringify(value ?? {}, null, 2)
    return formatted ?? '{}'
  }

  const status = $derived(readString('status'))
  const stream = $derived(readString('stream'))
  const phase = $derived(readString('phase'))
  const command = $derived(readString('command'))
  const target = $derived(readString('file', 'path', 'target'))
  const patch = $derived(readString('patch'))
  const payloadDetails = $derived(otherPayload())
</script>

<div class="border-border/70 bg-muted/20 space-y-2 rounded-2xl border px-3 py-2.5 text-sm">
  <div class="mb-1 text-[10px] font-semibold tracking-[0.16em] uppercase opacity-70">status</div>
  <div class="font-medium">{entry.title}</div>

  {#if entry.detail}
    <div class="text-muted-foreground text-xs leading-5 whitespace-pre-wrap">{entry.detail}</div>
  {/if}

  {#if status || stream || phase}
    <div class="flex flex-wrap gap-1.5 text-[11px]">
      {#if status}
        <span class="bg-background rounded-full border px-2 py-0.5 font-medium">{status}</span>
      {/if}
      {#if stream}
        <span class="bg-background rounded-full border px-2 py-0.5 font-medium">{stream}</span>
      {/if}
      {#if phase}
        <span class="bg-background rounded-full border px-2 py-0.5 font-medium">{phase}</span>
      {/if}
    </div>
  {/if}

  {#if command}
    <div class="bg-background/80 rounded-xl border px-3 py-2">
      <div class="mb-1 text-[10px] font-semibold tracking-[0.14em] uppercase opacity-70">
        command
      </div>
      <pre class="font-mono text-xs leading-5 whitespace-pre-wrap">{command}</pre>
    </div>
  {/if}

  {#if target}
    <div class="bg-background/80 rounded-xl border px-3 py-2">
      <div class="mb-1 text-[10px] font-semibold tracking-[0.14em] uppercase opacity-70">
        target
      </div>
      <div class="font-mono text-xs leading-5">{target}</div>
    </div>
  {/if}

  {#if patch}
    <details class="bg-background/80 rounded-xl border">
      <summary class="cursor-pointer px-3 py-2 text-xs font-medium">Patch preview</summary>
      <pre
        class="overflow-x-auto border-t px-3 py-2 font-mono text-xs leading-5 whitespace-pre-wrap">{patch}</pre>
    </details>
  {/if}

  {#if payloadDetails}
    <details class="bg-background/80 rounded-xl border">
      <summary class="cursor-pointer px-3 py-2 text-xs font-medium">Payload details</summary>
      <pre
        class="overflow-x-auto border-t px-3 py-2 font-mono text-xs leading-5 whitespace-pre-wrap">{formatJSON(
          payloadDetails,
        )}</pre>
    </details>
  {/if}
</div>
