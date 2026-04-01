<script lang="ts">
  import type { SkillFile } from '$lib/api/contracts'
  import { buildSkillTreeEntries, type SkillTreeKind } from './skill-bundle-editor'
  import { File, FileCode, FileText, FolderOpen, Package } from '@lucide/svelte'

  let {
    files,
    emptyDirectoryPaths = [],
    selectedPath,
    onSelect,
  }: {
    files: SkillFile[]
    emptyDirectoryPaths?: string[]
    selectedPath: string | null
    onSelect?: (path: string, kind: SkillTreeKind) => void
  } = $props()

  const tree = $derived(buildSkillTreeEntries(files, emptyDirectoryPaths))

  function fileIcon(file: SkillFile | undefined) {
    if (!file) return FolderOpen
    switch (file.file_kind) {
      case 'entrypoint':
        return Package
      case 'script':
        return FileCode
      case 'reference':
        return FileText
      default:
        return File
    }
  }

  function kindColor(file: SkillFile | undefined): string {
    if (!file) return 'text-muted-foreground'
    switch (file.file_kind) {
      case 'entrypoint':
        return 'text-amber-500'
      case 'script':
        return 'text-blue-500'
      case 'reference':
        return 'text-emerald-500'
      default:
        return 'text-muted-foreground'
    }
  }
</script>

<div class="flex flex-col text-sm" data-testid="skill-file-tree">
  {#each tree as node (node.path)}
    {@const Icon = fileIcon(node.file)}
    <button
      type="button"
      class="flex items-center gap-2 rounded-md px-2 py-1 text-left transition-colors
        {selectedPath === node.path
        ? 'bg-accent text-accent-foreground'
        : 'text-muted-foreground hover:bg-muted hover:text-foreground'}"
      style:padding-left={`${node.depth * 12 + 8}px`}
      onclick={() => onSelect?.(node.path, node.kind)}
    >
      <Icon class="size-3.5 shrink-0 {kindColor(node.file)}" />
      <span class="truncate text-xs {node.kind === 'directory' ? 'font-medium' : ''}">
        {node.kind === 'directory' ? `${node.name}/` : node.name}
      </span>
    </button>
  {/each}
</div>
