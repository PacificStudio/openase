<script lang="ts">
  import type { SkillFile } from '$lib/api/contracts'
  import { Badge } from '$ui/badge'
  import { i18nStore } from '$lib/i18n/store.svelte'

  let {
    fileCount,
    totalSizeLabel,
    selectedFile = null,
    dirtyCount = 0,
  }: {
    fileCount: number
    totalSizeLabel: string
    selectedFile?: SkillFile | null
    dirtyCount?: number
  } = $props()
</script>

<footer
  class="border-border bg-muted/30 text-muted-foreground flex shrink-0 items-center gap-4 border-t px-4 py-1 text-[11px]"
>
  <span>
    {fileCount} {i18nStore.t('skills.editorStatusBar.filesLabel')}
  </span>
  <span>{totalSizeLabel}</span>
  {#if selectedFile}
    <span>{selectedFile.file_kind}</span>
    <span>{selectedFile.media_type}</span>
    {#if selectedFile.is_executable}
    <Badge variant="outline" class="h-4 px-1 text-[9px]">
      {i18nStore.t('skills.editorStatusBar.executableLabel')}
    </Badge>
    {/if}
  {/if}
  {#if dirtyCount > 0}
    <span class="text-primary ml-auto font-medium">
      {dirtyCount} {i18nStore.t('skills.editorStatusBar.unsavedLabel')}
    </span>
  {/if}
</footer>
