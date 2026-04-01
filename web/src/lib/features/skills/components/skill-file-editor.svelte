<script lang="ts">
  import type { SkillFile } from '$lib/api/contracts'
  import { Textarea } from '$ui/textarea'
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
  <Textarea
    value={content}
    class="h-full min-h-0 flex-1 resize-none rounded-none border-0 font-mono text-sm leading-relaxed focus-visible:ring-0"
    oninput={(event) =>
      onContentChange?.(file.path, (event.currentTarget as HTMLTextAreaElement).value)}
  />
{/if}
