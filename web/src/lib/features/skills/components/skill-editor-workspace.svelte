<script lang="ts">
  import type { Skill, SkillFile, Workflow } from '$lib/api/contracts'
  import SkillFileEditor from './skill-file-editor.svelte'
  import SkillFileTabs from './skill-file-tabs.svelte'
  import SkillFileTree from './skill-file-tree.svelte'
  import type { SkillTreeKind } from './skill-bundle-editor'
  import SkillMetadataPanel from './skill-metadata-panel.svelte'
  import { FilePlus2, FolderPlus } from '@lucide/svelte'

  let {
    files,
    emptyDirectoryPaths = [],
    selectedPath,
    selectedTreePath,
    openFiles,
    dirtyPaths,
    selectedFile,
    activeContent,
    skill,
    workflows,
    history,
    busy = false,
    metadataOpen = true,
    pendingCreate = null,
    editDescription = $bindable(''),
    onSelectTreeNode,
    onCreateFile,
    onCreateFolder,
    onCreateCommit,
    onCreateCancel,
    onRenameNode,
    onDeleteNode,
    onSelectTab,
    onCloseTab,
    onContentChange,
    onToggleBinding,
  }: {
    files: SkillFile[]
    emptyDirectoryPaths?: string[]
    selectedPath: string | null
    selectedTreePath: string | null
    openFiles: SkillFile[]
    dirtyPaths: Set<string>
    selectedFile: SkillFile | null
    activeContent: string
    skill: Skill
    workflows: Workflow[]
    history: Array<{ id: string; version: number; created_by: string; created_at: string }>
    busy?: boolean
    metadataOpen?: boolean
    pendingCreate?: { kind: 'file' | 'folder'; parentPath: string } | null
    editDescription?: string
    onSelectTreeNode?: (path: string, kind: SkillTreeKind) => void
    onCreateFile?: () => void
    onCreateFolder?: () => void
    onCreateCommit?: (path: string, kind: 'file' | 'folder') => void
    onCreateCancel?: () => void
    onRenameNode?: (oldPath: string, newPath: string, kind: SkillTreeKind) => void
    onDeleteNode?: (path: string, kind: SkillTreeKind) => void
    onSelectTab?: (path: string) => void
    onCloseTab?: (path: string) => void
    onContentChange?: (path: string, value: string) => void
    onToggleBinding?: (workflowId: string, shouldBind: boolean) => void
  } = $props()
</script>

<div class="flex min-h-0 flex-1">
  <aside class="border-border w-48 shrink-0 overflow-y-auto border-r p-2">
    <div class="mb-1.5 flex items-center justify-between px-1">
      <span class="text-muted-foreground text-[11px] font-medium tracking-wider uppercase">
        Files
      </span>
      <div class="flex items-center gap-0.5">
        <button
          type="button"
          class="text-muted-foreground hover:text-foreground hover:bg-muted inline-flex size-5 items-center justify-center rounded transition-colors"
          title="New file"
          onclick={onCreateFile}
        >
          <FilePlus2 class="size-3.5" />
        </button>
        <button
          type="button"
          class="text-muted-foreground hover:text-foreground hover:bg-muted inline-flex size-5 items-center justify-center rounded transition-colors"
          title="New folder"
          onclick={onCreateFolder}
        >
          <FolderPlus class="size-3.5" />
        </button>
      </div>
    </div>

    <SkillFileTree
      {files}
      {emptyDirectoryPaths}
      selectedPath={selectedTreePath}
      {pendingCreate}
      onSelect={onSelectTreeNode}
      {onCreateCommit}
      {onCreateCancel}
      onRename={onRenameNode}
      onDelete={onDeleteNode}
    />
  </aside>

  <div class="flex min-w-0 flex-1 flex-col">
    <SkillFileTabs {openFiles} activeFile={selectedPath} {dirtyPaths} {onSelectTab} {onCloseTab} />

    <div class="min-h-0 flex-1 overflow-auto">
      <SkillFileEditor file={selectedFile} content={activeContent} {onContentChange} />
    </div>
  </div>

  {#if metadataOpen}
    <aside class="border-border w-64 shrink-0 overflow-y-auto border-l p-3">
      <SkillMetadataPanel
        {skill}
        {workflows}
        {busy}
        bind:editDescription
        {history}
        {onToggleBinding}
      />
    </aside>
  {/if}
</div>
