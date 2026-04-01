<script lang="ts">
  import { Badge } from '$ui/badge'
  import type { TicketRunTranscriptBlock } from '../types'

  let { block }: { block: Extract<TicketRunTranscriptBlock, { kind: 'interrupt' }> } = $props()

  function interruptString(...keys: string[]) {
    for (const key of keys) {
      const value = block.payload[key]
      if (typeof value === 'string' && value.trim()) {
        return value
      }
    }
    return ''
  }

  function interruptQuestion() {
    const questions = block.payload.questions
    if (!Array.isArray(questions) || questions.length === 0) {
      return ''
    }
    const first = questions[0]
    if (!first || typeof first !== 'object') {
      return ''
    }
    const value = (first as Record<string, unknown>).question
    return typeof value === 'string' ? value : ''
  }
</script>

<div class="space-y-3 text-sm">
  <p class="font-medium text-amber-950">{block.title}</p>
  <p class="text-amber-900">{block.summary}</p>

  {#if interruptString('command')}
    <div class="rounded-xl border border-amber-300/70 bg-white/80 px-3 py-2">
      <div class="mb-1 text-[10px] font-semibold tracking-[0.14em] text-amber-700 uppercase">
        command
      </div>
      <pre class="font-mono text-xs leading-5 whitespace-pre-wrap text-amber-950">{interruptString(
          'command',
        )}</pre>
    </div>
  {/if}

  {#if interruptString('file', 'path', 'target')}
    <div class="rounded-xl border border-amber-300/70 bg-white/80 px-3 py-2">
      <div class="mb-1 text-[10px] font-semibold tracking-[0.14em] text-amber-700 uppercase">
        target
      </div>
      <div class="font-mono text-xs leading-5 text-amber-950">
        {interruptString('file', 'path', 'target')}
      </div>
    </div>
  {/if}

  {#if interruptQuestion()}
    <div class="rounded-xl border border-amber-300/70 bg-white/80 px-3 py-2 text-amber-950">
      {interruptQuestion()}
    </div>
  {/if}

  {#if block.options.length > 0}
    <div class="flex flex-wrap gap-2">
      {#each block.options as option}
        <Badge variant="outline" class="border-amber-300 bg-white text-amber-900">
          {option.label}
        </Badge>
      {/each}
    </div>
  {/if}
</div>
