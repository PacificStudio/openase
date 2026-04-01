<script lang="ts">
  import { ChatMarkdownContent } from '$lib/features/chat'
  import { Button } from '$ui/button'
  import type { TicketRunTranscriptBlock } from '../types'

  const outputPreviewMaxLines = 14
  const outputPreviewMinChars = 720

  let {
    block,
    expanded = false,
    onToggle,
  }: {
    block: Extract<TicketRunTranscriptBlock, { kind: 'assistant_message' | 'terminal_output' }>
    expanded?: boolean
    onToggle?: () => Promise<void> | void
  } = $props()

  function outputLineCount(text: string) {
    return text.split('\n').length
  }

  function isExpandable(block: Extract<TicketRunTranscriptBlock, { kind: 'terminal_output' }>) {
    return (
      block.text.length >= outputPreviewMinChars ||
      outputLineCount(block.text) > outputPreviewMaxLines
    )
  }
</script>

{#if block.kind === 'assistant_message'}
  <div class="prose prose-sm prose-neutral max-w-none break-words">
    <ChatMarkdownContent source={block.text} />
  </div>
{:else}
  <div class="space-y-3">
    <div
      class={`overflow-x-auto rounded-xl border border-white/10 bg-black/20 px-3 py-3 font-mono text-xs leading-5 whitespace-pre-wrap ${
        isExpandable(block) && !expanded ? 'max-h-72 overflow-y-hidden' : ''
      }`}
    >
      {block.text}
    </div>
    {#if isExpandable(block)}
      <Button
        size="sm"
        variant="outline"
        class="h-8 border-white/15 bg-white/5 text-xs text-slate-100 hover:bg-white/10"
        onclick={() => void onToggle?.()}
      >
        {expanded ? 'Collapse output' : 'Expand output'}
      </Button>
    {/if}
  </div>
{/if}
