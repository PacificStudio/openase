<script lang="ts">
  import type { SkillFile } from '$lib/api/contracts'
  import { X } from '@lucide/svelte'

  let {
    openFiles,
    activeFile,
    dirtyPaths = new Set<string>(),
    onSelectTab,
    onCloseTab,
  }: {
    openFiles: SkillFile[]
    activeFile: string | null
    dirtyPaths?: Set<string>
    onSelectTab?: (path: string) => void
    onCloseTab?: (path: string) => void
  } = $props()

  function fileName(path: string) {
    return path.split('/').pop() ?? path
  }
</script>

{#if openFiles.length > 0}
  <div
    class="border-border bg-muted/30 flex items-center gap-0 overflow-x-auto border-b"
    data-testid="skill-file-tabs"
  >
    {#each openFiles as file (file.path)}
      {@const active = activeFile === file.path}
      {@const dirty = dirtyPaths.has(file.path)}
      <div
        class="group border-border/50 flex items-center border-r
          {active
          ? 'bg-background text-foreground'
          : 'text-muted-foreground hover:bg-background/50'}"
      >
        <button
          type="button"
          class="flex items-center gap-1.5 px-3 py-1.5 text-xs"
          onclick={() => onSelectTab?.(file.path)}
        >
          {#if dirty}
            <span class="bg-primary size-1.5 shrink-0 rounded-full"></span>
          {/if}
          <span class:font-medium={active}>{fileName(file.path)}</span>
        </button>
        <button
          type="button"
          class="mr-1 rounded p-0.5 opacity-0 transition-opacity group-hover:opacity-100
            {active ? 'opacity-60' : ''} hover:bg-muted"
          onclick={(e) => {
            e.stopPropagation()
            onCloseTab?.(file.path)
          }}
        >
          <X class="size-3" />
        </button>
      </div>
    {/each}
  </div>
{/if}
