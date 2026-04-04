<script lang="ts">
  import CollapsibleDiffCard from '$lib/components/collapsible-diff-card.svelte'
  import type { TicketRunTranscriptBlock } from '../types'

  let { block }: { block: Extract<TicketRunTranscriptBlock, { kind: 'diff' }> } = $props()
</script>

<CollapsibleDiffCard diff={block.diff} />
