<script lang="ts">
  import { cn } from '$lib/utils'
  import { LoaderCircle } from '@lucide/svelte'
  import { Button } from '$ui/button'
  import Textarea from '$ui/textarea/textarea.svelte'
  import EphemeralChatActionProposalCard from './ephemeral-chat-action-proposal-card.svelte'
  import EphemeralChatDiffCard from './ephemeral-chat-diff-card.svelte'
  import ChatMarkdownContent from './chat-markdown-content.svelte'
  import type { ProjectConversationTranscriptEntry } from './project-conversation-transcript-state'

  let {
    entries,
    pending = false,
    onConfirmActionProposal,
    onRespondInterrupt,
  }: {
    entries: ProjectConversationTranscriptEntry[]
    pending?: boolean
    onConfirmActionProposal?: (entryId: string) => Promise<void> | void
    onRespondInterrupt?: (input: {
      interruptId: string
      decision?: string
      answer?: Record<string, unknown>
    }) => Promise<void> | void
  } = $props()

  let interruptAnswers = $state<Record<string, string>>({})

  function firstQuestion(
    entry: Extract<ProjectConversationTranscriptEntry, { kind: 'interrupt' }>,
  ) {
    const questions = entry.payload.questions
    if (!Array.isArray(questions) || questions.length === 0) {
      return null
    }

    const question = questions[0]
    return question && typeof question === 'object' ? (question as Record<string, unknown>) : null
  }

  function questionOptions(
    entry: Extract<ProjectConversationTranscriptEntry, { kind: 'interrupt' }>,
  ) {
    const question = firstQuestion(entry)
    const options = question?.options
    if (!Array.isArray(options)) {
      return []
    }
    return options
      .map((item) => (item && typeof item === 'object' ? (item as Record<string, unknown>) : null))
      .filter((item): item is Record<string, unknown> => item != null)
  }

  function questionId(entry: Extract<ProjectConversationTranscriptEntry, { kind: 'interrupt' }>) {
    const question = firstQuestion(entry)
    const value = question?.id
    return typeof value === 'string' ? value : ''
  }

  function questionPrompt(
    entry: Extract<ProjectConversationTranscriptEntry, { kind: 'interrupt' }>,
  ) {
    const question = firstQuestion(entry)
    const value = question?.question
    return typeof value === 'string' ? value : 'Additional input is required to continue this turn.'
  }
</script>

<div class="space-y-3">
  {#each entries as entry (entry.id)}
    {#if entry.kind === 'action_proposal'}
      <EphemeralChatActionProposalCard
        {entry}
        onConfirm={onConfirmActionProposal}
        onCancel={undefined}
      />
    {:else if entry.kind === 'diff'}
      <EphemeralChatDiffCard {entry} />
    {:else if entry.kind === 'interrupt'}
      <div class="rounded-2xl border border-amber-300 bg-amber-50/70 px-3 py-3 text-sm">
        <div class="mb-1 text-[10px] font-semibold tracking-[0.16em] text-amber-700 uppercase">
          interrupt
        </div>
        <div class="mb-2 font-medium text-amber-900">
          {entry.interruptKind === 'user_input'
            ? 'User input required'
            : entry.interruptKind === 'file_change_approval'
              ? 'File change approval required'
              : 'Command approval required'}
        </div>

        {#if entry.status === 'resolved'}
          <div class="text-xs text-amber-800">
            Resolved{entry.decision ? `: ${entry.decision}` : '.'}
          </div>
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
        {:else if questionOptions(entry).length > 0}
          <div class="space-y-2">
            <div class="text-xs text-amber-800">{questionPrompt(entry)}</div>
            <div class="flex flex-wrap gap-2">
              {#each questionOptions(entry) as option}
                <Button
                  size="sm"
                  variant="outline"
                  class="border-amber-300 bg-white"
                  onclick={() =>
                    void onRespondInterrupt?.({
                      interruptId: entry.interruptId,
                      answer: {
                        [questionId(entry)]: { answers: [String(option.label ?? '')] },
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
            <div class="text-xs text-amber-800">{questionPrompt(entry)}</div>
            <Textarea
              bind:value={interruptAnswers[entry.interruptId]}
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
                    [questionId(entry) || 'answer']: {
                      answers: [interruptAnswers[entry.interruptId] ?? ''],
                    },
                  },
                })}
            >
              Submit
            </Button>
          </div>
        {/if}
      </div>
    {:else}
      <div
        class={cn(
          'rounded-2xl border px-3 py-2.5 text-sm leading-6',
          entry.role === 'user' && 'bg-primary text-primary-foreground',
          entry.role === 'assistant' && 'border-border bg-muted/40 text-foreground',
          entry.role === 'system' && 'border-border bg-muted/20 text-foreground',
        )}
      >
        <div class="mb-1 text-[10px] font-semibold tracking-[0.16em] uppercase opacity-70">
          {entry.role}
        </div>
        {#if entry.role === 'assistant'}
          <ChatMarkdownContent source={entry.content} />
        {:else}
          <div class="break-words whitespace-pre-wrap">{entry.content}</div>
        {/if}
      </div>
    {/if}
  {/each}

  {#if pending}
    <div
      class="border-border bg-muted/30 flex items-center gap-2 rounded-2xl border px-3 py-2.5 text-sm"
    >
      <LoaderCircle class="size-4 animate-spin" />
      Thinking…
    </div>
  {/if}
</div>
