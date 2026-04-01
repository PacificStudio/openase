<script lang="ts">
  import type { SkillFile } from '$lib/api/contracts'
  import { File, FileCode, FileText, FolderOpen, Package } from '@lucide/svelte'

  let {
    files,
    selectedPath,
    onSelect,
  }: {
    files: SkillFile[]
    selectedPath: string | null
    onSelect?: (path: string) => void
  } = $props()

  type TreeNode = {
    name: string
    path: string
    file?: SkillFile
    children: TreeNode[]
  }

  const tree = $derived.by(() => {
    const root: TreeNode = { name: '', path: '', children: [] }

    for (const file of files) {
      const parts = file.path.split('/')
      let current = root
      for (let i = 0; i < parts.length; i++) {
        const part = parts[i]
        const isFile = i === parts.length - 1
        let child = current.children.find((c) => c.name === part)
        if (!child) {
          child = {
            name: part,
            path: parts.slice(0, i + 1).join('/'),
            file: isFile ? file : undefined,
            children: [],
          }
          current.children.push(child)
        }
        if (isFile) {
          child.file = file
        }
        current = child
      }
    }

    // Sort: directories first, then alphabetically
    function sortChildren(node: TreeNode) {
      node.children.sort((a, b) => {
        const aIsDir = a.children.length > 0 && !a.file
        const bIsDir = b.children.length > 0 && !b.file
        if (aIsDir !== bIsDir) return aIsDir ? -1 : 1
        return a.name.localeCompare(b.name)
      })
      for (const child of node.children) {
        sortChildren(child)
      }
    }

    sortChildren(root)
    return root.children
  })

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
    {#if node.file}
      {@const Icon = fileIcon(node.file)}
      <button
        type="button"
        class="flex items-center gap-2 rounded-md px-2 py-1 text-left transition-colors
          {selectedPath === node.path
          ? 'bg-accent text-accent-foreground'
          : 'text-muted-foreground hover:bg-muted hover:text-foreground'}"
        onclick={() => onSelect?.(node.path)}
      >
        <Icon class="size-3.5 shrink-0 {kindColor(node.file)}" />
        <span class="truncate text-xs">{node.name}</span>
      </button>
    {:else}
      <div class="mt-2 first:mt-0">
        <div class="text-muted-foreground flex items-center gap-2 px-2 py-1">
          <FolderOpen class="size-3.5 shrink-0" />
          <span class="truncate text-xs font-medium">{node.name}/</span>
        </div>
        <div class="border-border/50 ml-3 border-l pl-1">
          {#each node.children as child (child.path)}
            {@const ChildIcon = fileIcon(child.file)}
            <button
              type="button"
              class="flex w-full items-center gap-2 rounded-md px-2 py-1 text-left transition-colors
                {selectedPath === child.path
                ? 'bg-accent text-accent-foreground'
                : 'text-muted-foreground hover:bg-muted hover:text-foreground'}"
              onclick={() => onSelect?.(child.path)}
            >
              <ChildIcon class="size-3.5 shrink-0 {kindColor(child.file)}" />
              <span class="truncate text-xs">{child.name}</span>
            </button>
          {/each}
        </div>
      </div>
    {/if}
  {/each}
</div>
