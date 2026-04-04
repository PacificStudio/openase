<script lang="ts">
  import CollapsibleDiffCard from '$lib/components/collapsible-diff-card.svelte'
  import type { ProjectConversationDiffEntry } from './project-conversation-transcript-types'

  let { entry }: { entry: ProjectConversationDiffEntry } = $props()
</script>

<article class="rounded-2xl border border-sky-500/20 bg-sky-500/5 px-4 py-3">
  <div class="mb-2 flex items-center justify-between gap-3 text-[11px]">
    <div class="flex min-w-0 items-center gap-2">
      <span class="text-muted-foreground font-medium uppercase tracking-[0.14em]">Diff</span>
      <span class="text-muted-foreground min-w-0 truncate">{entry.diff.file}</span>
    </div>
  </div>

  <CollapsibleDiffCard diff={entry.diff} />
</article>
