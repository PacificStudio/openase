<script lang="ts">
  import { Button } from '$ui/button'
  import CollapsibleDiffCard from '$lib/components/collapsible-diff-card.svelte'
  import { workspaceBrowserPortal } from './workspace-browser-portal.svelte'
  import type { ProjectConversationDiffEntry } from './project-conversation-transcript-types'

  let { entry }: { entry: ProjectConversationDiffEntry } = $props()
</script>

<div class="space-y-2">
  <CollapsibleDiffCard diff={entry.diff} />
  <div class="flex flex-wrap gap-2">
    <Button
      size="sm"
      variant="secondary"
      onclick={() => workspaceBrowserPortal.reviewPatch(entry.diff)}
    >
      Review patch
    </Button>
    <Button size="sm" onclick={() => workspaceBrowserPortal.reviewPatch(entry.diff, true)}>
      Apply patch to editor
    </Button>
  </div>
</div>
