<script lang="ts">
  import { cn } from '$lib/utils'
  import { Button } from '$ui/button'
  import Textarea from '$ui/textarea/textarea.svelte'
  import { ChevronRight, ShieldAlert, CheckCircle } from '@lucide/svelte'
  import type { ProjectConversationTranscriptEntry } from './project-conversation-transcript-state'

  let {
    entry,
    standalone = false,
    onRespondInterrupt,
  }: {
    entry: Extract<ProjectConversationTranscriptEntry, { kind: 'interrupt' }>
    standalone?: boolean
    onRespondInterrupt?: (input: {
      interruptId: string
      decision?: string
      answer?: Record<string, unknown>
    }) => Promise<void> | void
  } = $props()

  let expanded = $state(true)
  let answer = $state('')

  function firstQuestion() {
    const questions = entry.payload.questions
    if (!Array.isArray(questions) || questions.length === 0) return null
    const question = questions[0]
    return question && typeof question === 'object' ? (question as Record<string, unknown>) : null
  }

  function questionOptions() {
    const options = firstQuestion()?.options
    if (!Array.isArray(options)) return []
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

  function readInterruptString(...keys: string[]) {
    for (const key of keys) {
      const value = entry.payload[key]
      if (typeof value === 'string' && value.trim()) return value
    }
    return ''
  }

  function interruptTitle() {
    if (entry.interruptKind === 'user_input') return 'User input required'
    if (entry.interruptKind === 'file_change_approval') return 'File change approval'
    return 'Command approval'
  }

  const target = $derived(readInterruptString('file', 'path'))
  const command = $derived(readInterruptString('command'))
  const patch = $derived(readInterruptString('patch'))
  const isResolved = $derived(entry.status === 'resolved')
  const hasPayload = $derived(Object.keys(entry.payload).length > 0)
</script>

<div class={cn('group', standalone && 'border-border/60 bg-muted/20 rounded-lg border')}>
  <button
    type="button"
    class="hover:bg-muted/40 flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-left text-xs transition-colors"
    onclick={() => (expanded = !expanded)}
  >
    <ChevronRight
      class={cn(
        'text-muted-foreground size-3 shrink-0 transition-transform duration-150',
        expanded && 'rotate-90',
      )}
    />
    {#if isResolved}
      <CheckCircle class="size-3.5 shrink-0 text-emerald-500" />
    {:else}
      <ShieldAlert class="size-3.5 shrink-0 text-amber-500" />
    {/if}
    <span class="text-foreground min-w-0 flex-1 truncate">{interruptTitle()}</span>
    <span class="text-muted-foreground/60 shrink-0 text-[10px]">
      {isResolved ? 'resolved' : 'pending'}
    </span>
  </button>

  {#if expanded}
    <div class="border-border/40 ml-5 space-y-2 border-l pt-1 pb-2 pl-3">
      {#if command}
        <div>
          <div
            class="text-muted-foreground mb-0.5 text-[10px] font-medium tracking-wider uppercase"
          >
            command
          </div>
          <pre
            class="bg-muted/60 overflow-x-auto rounded-md px-2.5 py-1.5 font-mono text-xs leading-5 whitespace-pre-wrap">{command}</pre>
        </div>
      {/if}

      {#if target}
        <div>
          <div
            class="text-muted-foreground mb-0.5 text-[10px] font-medium tracking-wider uppercase"
          >
            target
          </div>
          <div class="text-foreground/80 font-mono text-xs">{target}</div>
        </div>
      {/if}

      {#if patch}
        <details>
          <summary class="text-muted-foreground hover:text-foreground cursor-pointer text-xs">
            Patch preview
          </summary>
          <pre
            class="bg-muted/60 mt-1 max-h-60 overflow-auto rounded-md px-2.5 py-1.5 font-mono text-xs leading-5 whitespace-pre-wrap">{patch}</pre>
        </details>
      {/if}

      {#if hasPayload}
        <details class="text-xs">
          <summary class="text-muted-foreground hover:text-foreground cursor-pointer">
            Payload details
          </summary>
          <pre
            class="bg-muted/60 mt-1 max-h-48 overflow-auto rounded-md px-2.5 py-1.5 font-mono text-[11px] leading-5 whitespace-pre-wrap">{JSON.stringify(
              entry.payload,
              null,
              2,
            )}</pre>
        </details>
      {/if}

      {#if isResolved}
        <div class="text-muted-foreground text-xs">
          Resolved{entry.decision ? `: ${entry.decision}` : '.'}
        </div>
      {:else if entry.options.length > 0}
        <div class="flex flex-wrap gap-1.5">
          {#each entry.options as option}
            <Button
              size="sm"
              variant="outline"
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
        <div class="space-y-1.5">
          <div class="text-muted-foreground text-xs">{questionPrompt()}</div>
          <div class="flex flex-wrap gap-1.5">
            {#each questionOptions() as option}
              <Button
                size="sm"
                variant="outline"
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
        <div class="space-y-1.5">
          <div class="text-muted-foreground text-xs">{questionPrompt()}</div>
          <Textarea
            bind:value={answer}
            rows={2}
            class="min-h-0 resize-none text-sm"
            placeholder="Enter your answer…"
          />
          <Button
            size="sm"
            variant="outline"
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
  {/if}
</div>
