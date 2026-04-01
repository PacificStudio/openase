<script lang="ts">
  import type { Skill, SkillFile, Workflow } from '$lib/api/contracts'
  import { Button } from '$ui/button'
  import SkillFileEditor from './skill-file-editor.svelte'
  import SkillFileTabs from './skill-file-tabs.svelte'
  import SkillFileTree from './skill-file-tree.svelte'
  import type { SkillTreeKind } from './skill-bundle-editor'
  import SkillMetadataPanel from './skill-metadata-panel.svelte'
  import { FilePlus2, FolderPlus, Pencil, Trash2 } from '@lucide/svelte'

  let {
    files,
    emptyDirectoryPaths = [],
    selectedPath,
    selectedTreePath,
    selectedTreeKind = null,
    openFiles,
    dirtyPaths,
    selectedFile,
    activeContent,
    skill,
    workflows,
    history,
    editing = false,
    busy = false,
    metadataOpen = true,
    editDescription = $bindable(''),
    onSelectTreeNode,
    onCreateFile,
    onCreateFolder,
    onRenameSelection,
    onDeleteSelection,
    onSelectTab,
    onCloseTab,
    onContentChange,
    onToggleBinding,
  }: {
    files: SkillFile[]
    emptyDirectoryPaths?: string[]
    selectedPath: string | null
    selectedTreePath: string | null
    selectedTreeKind?: SkillTreeKind | null
    openFiles: SkillFile[]
    dirtyPaths: Set<string>
    selectedFile: SkillFile | null
    activeContent: string
    skill: Skill
    workflows: Workflow[]
    history: Array<{ id: string; version: number; created_by: string; created_at: string }>
    editing?: boolean
    busy?: boolean
    metadataOpen?: boolean
    editDescription?: string
    onSelectTreeNode?: (path: string, kind: SkillTreeKind) => void
    onCreateFile?: () => void
    onCreateFolder?: () => void
    onRenameSelection?: () => void
    onDeleteSelection?: () => void
    onSelectTab?: (path: string) => void
    onCloseTab?: (path: string) => void
    onContentChange?: (path: string, value: string) => void
    onToggleBinding?: (workflowId: string, shouldBind: boolean) => void
  } = $props()
</script>

<div class="flex min-h-0 flex-1">
  <aside class="border-border w-48 shrink-0 overflow-y-auto border-r p-2">
    {#if editing}
      <div class="mb-2 grid grid-cols-2 gap-1">
        <Button
          size="sm"
          variant="outline"
          class="h-7 gap-1 px-2 text-[11px]"
          onclick={onCreateFile}
        >
          <FilePlus2 class="size-3.5" />
          File
        </Button>
        <Button
          size="sm"
          variant="outline"
          class="h-7 gap-1 px-2 text-[11px]"
          onclick={onCreateFolder}
        >
          <FolderPlus class="size-3.5" />
          Folder
        </Button>
        <Button
          size="sm"
          variant="outline"
          class="h-7 gap-1 px-2 text-[11px]"
          onclick={onRenameSelection}
          disabled={!selectedTreePath}
        >
          <Pencil class="size-3.5" />
          Rename
        </Button>
        <Button
          size="sm"
          variant="outline"
          class="text-destructive hover:text-destructive h-7 gap-1 px-2 text-[11px]"
          onclick={onDeleteSelection}
          disabled={!selectedTreePath}
        >
          <Trash2 class="size-3.5" />
          Delete
        </Button>
      </div>
      <div class="text-muted-foreground mb-2 min-h-8 px-1 text-[11px] leading-4">
        {#if selectedTreePath}
          <span class="font-medium">{selectedTreeKind === 'directory' ? 'Folder' : 'File'}:</span>
          {selectedTreePath}
        {:else}
          Select a file or folder to rename or delete it.
        {/if}
      </div>
    {/if}

    <SkillFileTree
      {files}
      {emptyDirectoryPaths}
      selectedPath={selectedTreePath}
      onSelect={onSelectTreeNode}
    />
  </aside>

  <div class="flex min-w-0 flex-1 flex-col">
    <SkillFileTabs {openFiles} activeFile={selectedPath} {dirtyPaths} {onSelectTab} {onCloseTab} />

    <div class="min-h-0 flex-1 overflow-auto">
      <SkillFileEditor file={selectedFile} content={activeContent} {editing} {onContentChange} />
    </div>
  </div>

  {#if metadataOpen}
    <aside class="border-border w-64 shrink-0 overflow-y-auto border-l p-3">
      <SkillMetadataPanel
        {skill}
        {workflows}
        {editing}
        {busy}
        bind:editDescription
        {history}
        {onToggleBinding}
      />
    </aside>
  {/if}
</div>
