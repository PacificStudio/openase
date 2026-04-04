<script lang="ts">
  import CollapsibleDiffCard from '$lib/components/collapsible-diff-card.svelte'
  import type { ProjectConversationDiffEntry } from './project-conversation-transcript-types'

  let { entry }: { entry: ProjectConversationDiffEntry } = $props()
</script>

<CollapsibleDiffCard diff={entry.diff} />
