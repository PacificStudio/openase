<script lang="ts">
  import type { EphemeralChatDiffEntry } from './transcript'

  let { entry }: { entry: EphemeralChatDiffEntry } = $props()

  const lineCount = $derived(entry.diff.hunks.reduce((count, hunk) => count + hunk.lines.length, 0))
</script>

<div class="space-y-2 rounded-2xl border border-sky-500/30 bg-sky-500/8 p-3 text-sm">
  <div class="text-[10px] font-semibold tracking-[0.16em] uppercase opacity-70">assistant</div>
  <div class="font-medium">Structured Diff</div>
  <p class="text-muted-foreground text-xs leading-5">
    Target: <span class="font-mono">{entry.diff.file}</span>. {entry.diff.hunks.length} hunk{entry
      .diff.hunks.length === 1
      ? ''
      : 's'} / {lineCount} line operation{lineCount === 1 ? '' : 's'}.
  </p>
</div>
