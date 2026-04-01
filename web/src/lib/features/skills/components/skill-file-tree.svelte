<script lang="ts">
  import type { SkillFile } from '$lib/api/contracts'
  import { buildSkillTreeEntries, type SkillTreeKind } from './skill-bundle-editor'
  import {
    File,
    FileCode,
    FileText,
    FolderOpen,
    Package,
    Ellipsis,
    Pencil,
    Trash2,
  } from '@lucide/svelte'
  import * as DropdownMenu from '$ui/dropdown-menu'
  import { cn } from '$lib/utils'

  let {
    files,
    emptyDirectoryPaths = [],
    selectedPath,
    pendingCreate = null,
    onSelect,
    onCreateCommit,
    onCreateCancel,
    onRename,
    onDelete,
  }: {
    files: SkillFile[]
    emptyDirectoryPaths?: string[]
    selectedPath: string | null
    pendingCreate?: { kind: 'file' | 'folder'; parentPath: string } | null
    onSelect?: (path: string, kind: SkillTreeKind) => void
    onCreateCommit?: (path: string, kind: 'file' | 'folder') => void
    onCreateCancel?: () => void
    onRename?: (oldPath: string, newPath: string, kind: SkillTreeKind) => void
    onDelete?: (path: string, kind: SkillTreeKind) => void
  } = $props()

  const tree = $derived(buildSkillTreeEntries(files, emptyDirectoryPaths))

  let renamingPath = $state<string | null>(null)
  let renamingKind = $state<SkillTreeKind>('file')
  let inlineValue = $state('')

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

  function startRename(path: string, kind: SkillTreeKind) {
    renamingPath = path
    renamingKind = kind
    inlineValue = path.split('/').pop() ?? path
  }

  function focusInlineInput(el: HTMLInputElement) {
    el.focus()
    if (renamingPath) {
      const name = inlineValue
      const dotIdx = renamingKind === 'file' ? name.lastIndexOf('.') : -1
      el.setSelectionRange(0, dotIdx > 0 ? dotIdx : name.length)
    } else {
      el.select()
    }
  }

  function commitRename() {
    if (!renamingPath || !inlineValue.trim()) {
      cancelInline()
      return
    }
    const parts = renamingPath.split('/')
    parts[parts.length - 1] = inlineValue.trim()
    const newPath = parts.join('/')
    const oldPath = renamingPath
    const kind = renamingKind
    renamingPath = null
    inlineValue = ''
    if (newPath !== oldPath) {
      onRename?.(oldPath, newPath, kind)
    }
  }

  function commitCreate() {
    if (!pendingCreate || !inlineValue.trim()) {
      onCreateCancel?.()
      inlineValue = ''
      return
    }
    const parent = pendingCreate.parentPath
    const fullPath = parent ? `${parent}/${inlineValue.trim()}` : inlineValue.trim()
    const kind = pendingCreate.kind
    inlineValue = ''
    onCreateCommit?.(fullPath, kind)
  }

  function cancelInline() {
    renamingPath = null
    inlineValue = ''
    onCreateCancel?.()
  }

  function handleInlineKeydown(event: KeyboardEvent) {
    if (event.isComposing) return
    if (event.key === 'Enter') {
      event.preventDefault()
      if (renamingPath) commitRename()
      else commitCreate()
    } else if (event.key === 'Escape') {
      event.preventDefault()
      cancelInline()
    }
  }

  function handleInlineBlur() {
    if (renamingPath) commitRename()
    else if (pendingCreate) commitCreate()
  }

  // Determine the depth for the pending create input
  const createInputDepth = $derived.by(() => {
    if (!pendingCreate) return 0
    if (!pendingCreate.parentPath) return 0
    return pendingCreate.parentPath.split('/').length
  })

  // Find the index after which to show the create input
  const createInsertAfterIndex = $derived.by(() => {
    if (!pendingCreate) return -1
    const parentPath = pendingCreate.parentPath

    if (!parentPath) {
      // Root level - show at end
      return tree.length - 1
    }

    // Find last node that is under parentPath
    let lastIndex = -1
    for (let i = 0; i < tree.length; i++) {
      if (tree[i].path === parentPath || tree[i].path.startsWith(`${parentPath}/`)) {
        lastIndex = i
      }
    }
    return lastIndex
  })

  const isEntrypoint = (path: string) => path === 'SKILL.md'
</script>

<div class="flex flex-col text-sm" data-testid="skill-file-tree">
  {#each tree as node, index (node.path)}
    {@const Icon = fileIcon(node.file)}
    {#if renamingPath === node.path}
      <!-- Inline rename input -->
      <div
        class="flex items-center gap-2 rounded-md px-2 py-1"
        style:padding-left={`${node.depth * 12 + 8}px`}
      >
        <Icon class="size-3.5 shrink-0 {kindColor(node.file)}" />
        <input
          type="text"
          class="bg-background border-primary min-w-0 flex-1 rounded border px-1.5 py-0.5 text-xs outline-none"
          bind:value={inlineValue}
          use:focusInlineInput
          onkeydown={handleInlineKeydown}
          onblur={handleInlineBlur}
        />
      </div>
    {:else}
      <!-- Normal tree node -->
      <div
        class={cn(
          'group relative flex items-center rounded-md transition-colors',
          selectedPath === node.path
            ? 'bg-accent text-accent-foreground'
            : 'text-muted-foreground hover:bg-muted hover:text-foreground',
        )}
      >
        <button
          type="button"
          class="flex min-w-0 flex-1 items-center gap-2 py-1 pr-6"
          style:padding-left={`${node.depth * 12 + 8}px`}
          onclick={() => onSelect?.(node.path, node.kind)}
        >
          <Icon class="size-3.5 shrink-0 {kindColor(node.file)}" />
          <span class="truncate text-xs {node.kind === 'directory' ? 'font-medium' : ''}">
            {node.kind === 'directory' ? `${node.name}/` : node.name}
          </span>
        </button>

        {#if !isEntrypoint(node.path)}
          <div
            class="absolute top-1/2 right-0.5 -translate-y-1/2 opacity-0 group-hover:opacity-100"
          >
            <DropdownMenu.Root>
              <DropdownMenu.Trigger
                class="text-muted-foreground hover:text-foreground hover:bg-muted inline-flex size-5 items-center justify-center rounded transition-colors"
                onclick={(e: MouseEvent) => e.stopPropagation()}
              >
                <Ellipsis class="size-3" />
              </DropdownMenu.Trigger>
              <DropdownMenu.Content align="end" class="w-32">
                <DropdownMenu.Item
                  class="gap-2 text-xs"
                  onclick={() => startRename(node.path, node.kind)}
                >
                  <Pencil class="size-3" />
                  Rename
                </DropdownMenu.Item>
                <DropdownMenu.Separator />
                <DropdownMenu.Item
                  class="text-destructive gap-2 text-xs"
                  onclick={() => onDelete?.(node.path, node.kind)}
                >
                  <Trash2 class="size-3" />
                  Delete
                </DropdownMenu.Item>
              </DropdownMenu.Content>
            </DropdownMenu.Root>
          </div>
        {/if}
      </div>
    {/if}

    <!-- Inline create input after the correct parent -->
    {#if pendingCreate && createInsertAfterIndex === index}
      <div
        class="flex items-center gap-2 rounded-md px-2 py-1"
        style:padding-left={`${createInputDepth * 12 + 8}px`}
      >
        {#if pendingCreate.kind === 'folder'}
          <FolderOpen class="text-muted-foreground size-3.5 shrink-0" />
        {:else}
          <File class="text-muted-foreground size-3.5 shrink-0" />
        {/if}
        <input
          type="text"
          class="bg-background border-primary min-w-0 flex-1 rounded border px-1.5 py-0.5 text-xs outline-none"
          bind:value={inlineValue}
          use:focusInlineInput
          onkeydown={handleInlineKeydown}
          onblur={handleInlineBlur}
        />
      </div>
    {/if}
  {/each}

  <!-- Create input when tree is empty or creating at root after all items -->
  {#if pendingCreate && (tree.length === 0 || (createInsertAfterIndex === -1 && !pendingCreate.parentPath))}
    <div class="flex items-center gap-2 rounded-md px-2 py-1" style:padding-left="8px">
      {#if pendingCreate.kind === 'folder'}
        <FolderOpen class="text-muted-foreground size-3.5 shrink-0" />
      {:else}
        <File class="text-muted-foreground size-3.5 shrink-0" />
      {/if}
      <input
        type="text"
        class="bg-background border-primary min-w-0 flex-1 rounded border px-1.5 py-0.5 text-xs outline-none"
        bind:value={inlineValue}
        use:focusInlineInput
        onkeydown={handleInlineKeydown}
        onblur={handleInlineBlur}
      />
    </div>
  {/if}
</div>
