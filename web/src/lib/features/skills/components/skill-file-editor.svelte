<script lang="ts">
  import type { SkillFile } from '$lib/api/contracts'
  import { CodeEditor } from '$lib/components/code'
  import { FileWarning } from '@lucide/svelte'

  let {
    file,
    content,
    onContentChange,
  }: {
    file: SkillFile | null
    content: string
    onContentChange?: (path: string, value: string) => void
  } = $props()

  const isText = $derived(file?.encoding === 'utf8')
  const isBinary = $derived(file ? !isText : false)
</script>

{#if !file}
  <div class="text-muted-foreground flex h-full flex-col items-center justify-center gap-2 text-sm">
    <p>Select a file from the tree to view or edit it.</p>
  </div>
{:else if isBinary}
  <div class="text-muted-foreground flex h-full flex-col items-center justify-center gap-3 text-sm">
    <FileWarning class="size-8 opacity-50" />
    <p>Binary file ({file.media_type})</p>
    <p class="text-xs">{file.size_bytes.toLocaleString()} bytes</p>
  </div>
{:else}
  <CodeEditor
    value={content}
    filePath={file.path}
    onchange={(value) => onContentChange?.(file.path, value)}
    class="h-full min-h-0 flex-1"
  />
{/if}
