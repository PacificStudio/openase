<script lang="ts">
  import { Button } from '$ui/button'
  import Textarea from '$ui/textarea/textarea.svelte'
  import type { ProjectConversationTranscriptEntry } from './project-conversation-transcript-state'

  let {
    entry,
    onRespondInterrupt,
  }: {
    entry: Extract<ProjectConversationTranscriptEntry, { kind: 'interrupt' }>
    onRespondInterrupt?: (input: {
      interruptId: string
      decision?: string
      answer?: Record<string, unknown>
    }) => Promise<void> | void
  } = $props()

  let answer = $state('')

  function firstQuestion() {
    const questions = entry.payload.questions
    if (!Array.isArray(questions) || questions.length === 0) {
      return null
    }

    const question = questions[0]
    return question && typeof question === 'object' ? (question as Record<string, unknown>) : null
  }

  function questionOptions() {
    const options = firstQuestion()?.options
    if (!Array.isArray(options)) {
      return []
    }
    return options
      .map((item) => (item && typeof item === 'object' ? (item as Record<string, unknown>) : null))
      .filter((item): item is Record<string, unknown> => item != null)
  }

  function questionId() {
    const value = firstQuestion()?.id
    return typeof value === 'string' ? value : ''
  }

  function questionPrompt() {
    const value = firstQuestion()?.question
    return typeof value === 'string' ? value : 'Additional input is required to continue this turn.'
  }

  function formatJSON(value: unknown) {
    const formatted = JSON.stringify(value ?? {}, null, 2)
    return formatted ?? '{}'
  }

  function readInterruptString(...keys: string[]) {
    for (const key of keys) {
      const value = entry.payload[key]
      if (typeof value === 'string' && value.trim()) {
        return value
      }
    }
    return ''
  }

  function interruptTitle() {
    if (entry.interruptKind === 'user_input') {
      return 'User input required'
    }
    if (entry.interruptKind === 'file_change_approval') {
      return 'File change approval required'
    }
    return 'Command approval required'
  }

  function interruptTarget() {
    return readInterruptString('file', 'path')
  }

  function interruptCommand() {
    return readInterruptString('command')
  }

  function interruptPatch() {
    return readInterruptString('patch')
  }

  function hasInterruptPayload() {
    return Object.keys(entry.payload).length > 0
  }
</script>

<div class="rounded-2xl border border-amber-300 bg-amber-50/70 px-3 py-3 text-sm">
  <div class="mb-1 text-[10px] font-semibold tracking-[0.16em] text-amber-700 uppercase">
    interrupt
  </div>
  <div class="mb-2 font-medium text-amber-900">{interruptTitle()}</div>

  {#if interruptCommand()}
    <div class="mb-2 rounded-xl border border-amber-300/70 bg-white/80 px-3 py-2">
      <div class="mb-1 text-[10px] font-semibold tracking-[0.14em] text-amber-700 uppercase">
        command
      </div>
      <pre
        class="font-mono text-xs leading-5 whitespace-pre-wrap text-amber-950">{interruptCommand()}</pre>
    </div>
  {/if}

  {#if interruptTarget()}
    <div class="mb-2 rounded-xl border border-amber-300/70 bg-white/80 px-3 py-2">
      <div class="mb-1 text-[10px] font-semibold tracking-[0.14em] text-amber-700 uppercase">
        target
      </div>
      <div class="font-mono text-xs leading-5 text-amber-950">{interruptTarget()}</div>
    </div>
  {/if}

  {#if interruptPatch()}
    <details class="mb-2 rounded-xl border border-amber-300/70 bg-white/80">
      <summary class="cursor-pointer px-3 py-2 text-xs font-medium text-amber-900">
        Patch preview
      </summary>
      <pre
        class="overflow-x-auto border-t border-amber-300/70 px-3 py-2 font-mono text-xs leading-5 whitespace-pre-wrap text-amber-950">
{interruptPatch()}</pre>
    </details>
  {/if}

  {#if hasInterruptPayload()}
    <details class="mb-2 rounded-xl border border-amber-300/70 bg-white/80">
      <summary class="cursor-pointer px-3 py-2 text-xs font-medium text-amber-900">
        Payload details
      </summary>
      <pre
        class="overflow-x-auto border-t border-amber-300/70 px-3 py-2 font-mono text-xs leading-5 whitespace-pre-wrap text-amber-950">
{formatJSON(entry.payload)}</pre>
    </details>
  {/if}

  {#if entry.status === 'resolved'}
    <div class="text-xs text-amber-800">Resolved{entry.decision ? `: ${entry.decision}` : '.'}</div>
  {:else if entry.options.length > 0}
    <div class="flex flex-wrap gap-2">
      {#each entry.options as option}
        <Button
          size="sm"
          variant="outline"
          class="border-amber-300 bg-white"
          onclick={() =>
            void onRespondInterrupt?.({
              interruptId: entry.interruptId,
              decision: option.id,
            })}
        >
          {option.label}
        </Button>
      {/each}
    </div>
  {:else if questionOptions().length > 0}
    <div class="space-y-2">
      <div class="text-xs text-amber-800">{questionPrompt()}</div>
      <div class="flex flex-wrap gap-2">
        {#each questionOptions() as option}
          <Button
            size="sm"
            variant="outline"
            class="border-amber-300 bg-white"
            onclick={() =>
              void onRespondInterrupt?.({
                interruptId: entry.interruptId,
                answer: {
                  [questionId()]: { answers: [String(option.label ?? '')] },
                },
              })}
          >
            {String(option.label ?? '')}
          </Button>
        {/each}
      </div>
    </div>
  {:else}
    <div class="space-y-2">
      <div class="text-xs text-amber-800">{questionPrompt()}</div>
      <Textarea
        bind:value={answer}
        rows={3}
        class="min-h-0 resize-none bg-white text-sm"
        placeholder="Enter your answer…"
      />
      <Button
        size="sm"
        variant="outline"
        class="border-amber-300 bg-white"
        onclick={() =>
          void onRespondInterrupt?.({
            interruptId: entry.interruptId,
            answer: {
              [questionId() || 'answer']: {
                answers: [answer],
              },
            },
          })}
      >
        Submit
      </Button>
    </div>
  {/if}
</div>
