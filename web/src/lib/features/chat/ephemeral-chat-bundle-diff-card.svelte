<script lang="ts">
  import type { EphemeralChatBundleDiffEntry } from './transcript'

  let { entry }: { entry: EphemeralChatBundleDiffEntry } = $props()

  const fileCount = $derived(entry.bundleDiff.files.length)
  const lineCount = $derived(
    entry.bundleDiff.files.reduce(
      (count, file) => count + file.hunks.reduce((sum, hunk) => sum + hunk.lines.length, 0),
      0,
    ),
  )
</script>

<div class="space-y-2 rounded-2xl border border-sky-500/30 bg-sky-500/8 p-3 text-sm">
  <div class="text-[10px] font-semibold tracking-[0.16em] uppercase opacity-70">assistant</div>
  <div class="font-medium">Structured Bundle Diff</div>
  <p class="text-muted-foreground text-xs leading-5">
    {fileCount} file{fileCount === 1 ? '' : 's'} / {lineCount} line operation{lineCount === 1
      ? ''
      : 's'}.
  </p>
</div>
