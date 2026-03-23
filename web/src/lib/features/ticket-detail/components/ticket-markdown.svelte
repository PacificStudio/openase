<script lang="ts">
  import { renderTicketCommentMarkdown } from '../markdown'

  let { body }: { body: string } = $props()
</script>

<div class="ticket-markdown text-sm leading-6 break-words">
  <!-- Safe here because renderTicketCommentMarkdown escapes HTML and sanitizes links. -->
  <!-- eslint-disable-next-line svelte/no-at-html-tags -->
  {@html renderTicketCommentMarkdown(body)}
</div>

<style>
  :global(.ticket-markdown p) {
    margin: 0.5rem 0 0;
  }

  :global(.ticket-markdown p:first-child) {
    margin-top: 0;
  }

  :global(.ticket-markdown ul),
  :global(.ticket-markdown ol) {
    margin: 0.5rem 0 0;
    padding-left: 1.25rem;
  }

  :global(.ticket-markdown pre) {
    overflow-x: auto;
    border-radius: 0.75rem;
    margin-top: 0.75rem;
    padding: 0.75rem;
    background: color-mix(in oklab, var(--color-muted) 72%, transparent);
  }

  :global(.ticket-markdown code) {
    font-family: var(--font-mono, monospace);
    font-size: 0.875em;
  }

  :global(.ticket-markdown a) {
    text-decoration: underline;
    text-underline-offset: 2px;
  }
</style>
