<script lang="ts">
  import type { Skill, SkillFile, Workflow } from '$lib/api/contracts'
  import SkillFileEditor from './skill-file-editor.svelte'
  import SkillFileTabs from './skill-file-tabs.svelte'
  import SkillFileTree from './skill-file-tree.svelte'
  import SkillMetadataPanel from './skill-metadata-panel.svelte'

  let {
    files,
    selectedPath,
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
    onSelectFile,
    onSelectTab,
    onCloseTab,
    onContentChange,
    onToggleBinding,
  }: {
    files: SkillFile[]
    selectedPath: string | null
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
    onSelectFile?: (path: string) => void
    onSelectTab?: (path: string) => void
    onCloseTab?: (path: string) => void
    onContentChange?: (path: string, value: string) => void
    onToggleBinding?: (workflowId: string, shouldBind: boolean) => void
  } = $props()
</script>

<div class="flex min-h-0 flex-1">
  <aside class="border-border w-48 shrink-0 overflow-y-auto border-r p-2">
    <SkillFileTree {files} {selectedPath} onSelect={onSelectFile} />
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
