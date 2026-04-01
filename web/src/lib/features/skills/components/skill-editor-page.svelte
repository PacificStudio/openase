<script lang="ts">
  import { goto, beforeNavigate } from '$app/navigation'
  import { ApiError } from '$lib/api/client'
  import type { SkillFile, Skill, Workflow } from '$lib/api/contracts'
  import {
    bindSkill,
    deleteSkill,
    disableSkill,
    enableSkill,
    unbindSkill,
    updateSkill,
  } from '$lib/api/openase'
  import { appStore } from '$lib/stores/app.svelte'
  import { projectPath } from '$lib/stores/app-context'
  import { toastStore } from '$lib/stores/toast.svelte'
  import SkillEditorHeader from './skill-editor-header.svelte'
  import {
    formatBytes,
    loadSkillEditorData,
    type SkillEditorHistoryEntry,
    selectInitialSkillFiles,
  } from './skill-editor-page.helpers'
  import {
    addEmptyDirectory,
    addDraftTextFile,
    cloneSkillFile,
    computeDirtyPaths,
    deleteDirectoryPath,
    deleteFilePath,
    encodeUTF8Base64,
    listEmptyDirectories,
    normalizeSkillBundlePath,
    renameDirectoryPath,
    renameFilePath,
    type SkillTreeKind,
    updateDraftTextFileContent,
  } from './skill-bundle-editor'
  import SkillEditorStatusBar from './skill-editor-status-bar.svelte'
  import SkillEditorWorkspace from './skill-editor-workspace.svelte'

  let { skillId }: { skillId: string } = $props()

  let skill = $state<Skill | null>(null)
  let files = $state<SkillFile[]>([])
  let draftFiles = $state<SkillFile[]>([])
  let emptyDirectoryPaths = $state<string[]>([])
  let history = $state<SkillEditorHistoryEntry[]>([])
  let workflows = $state<Workflow[]>([])

  let loading = $state(true)
  let busy = $state(false)
  let editDescription = $state('')
  let metadataOpen = $state(true)

  let selectedFilePath = $state<string | null>(null)
  let selectedTreePath = $state<string | null>(null)
  let selectedTreeKind = $state<SkillTreeKind | null>(null)
  let openFilePaths = $state<string[]>([])

  let pendingCreate = $state<{ kind: 'file' | 'folder'; parentPath: string } | null>(null)

  const dirtyPaths = $derived(computeDirtyPaths(files, draftFiles))
  const descriptionDirty = $derived(skill ? editDescription.trim() !== skill.description : false)
  const hasDirtyChanges = $derived(dirtyPaths.size > 0 || descriptionDirty)
  const emptyDraftDirectories = $derived(listEmptyDirectories(draftFiles, emptyDirectoryPaths))

  const selectedFile = $derived(
    selectedFilePath ? (draftFiles.find((f) => f.path === selectedFilePath) ?? null) : null,
  )

  const openFiles = $derived(
    openFilePaths
      .map((p) => draftFiles.find((f) => f.path === p))
      .filter((f): f is SkillFile => f !== undefined),
  )

  const activeContent = $derived(selectedFile?.content ?? '')

  const fileCount = $derived(draftFiles.length)
  const totalSize = $derived(draftFiles.reduce((sum, f) => sum + f.size_bytes, 0))

  // Load skill data
  $effect(() => {
    if (!skillId) return

    let cancelled = false
    loading = true

    const projectId = appStore.currentProject?.id

    void (async () => {
      try {
        const loaded = await loadSkillEditorData(skillId, projectId)

        if (cancelled) return

        skill = loaded.skill
        files = loaded.files
        draftFiles = loaded.files.map(cloneSkillFile)
        emptyDirectoryPaths = []
        history = loaded.history
        workflows = loaded.workflows
        editDescription = loaded.skill.description

        const selection = selectInitialSkillFiles(loaded.files)
        selectedFilePath = selection.selectedFilePath
        selectedTreePath = selection.selectedFilePath
        selectedTreeKind = selection.selectedFilePath ? 'file' : null
        openFilePaths = selection.openFilePaths
      } catch (err) {
        if (!cancelled) {
          toastStore.error(err instanceof ApiError ? err.detail : 'Failed to load skill.')
        }
      } finally {
        if (!cancelled) {
          loading = false
        }
      }
    })()

    return () => {
      cancelled = true
    }
  })

  // Ctrl+S to save
  function handleKeydown(event: KeyboardEvent) {
    if ((event.metaKey || event.ctrlKey) && event.key === 's') {
      event.preventDefault()
      if (!busy && hasDirtyChanges) {
        void handleSave()
      }
    }
  }

  // Warn on browser tab close with unsaved changes
  function handleBeforeUnload(event: BeforeUnloadEvent) {
    if (hasDirtyChanges) {
      event.preventDefault()
    }
  }

  // Warn on SvelteKit navigation with unsaved changes
  beforeNavigate(({ cancel }) => {
    if (hasDirtyChanges) {
      if (!window.confirm('You have unsaved changes. Leave without saving?')) {
        cancel()
      }
    }
  })

  function navigateBack() {
    const orgId = appStore.currentOrg?.id
    const projectId = appStore.currentProject?.id
    if (orgId && projectId) {
      void goto(projectPath(orgId, projectId, 'skills'))
    }
  }

  function selectFile(path: string) {
    selectedFilePath = path
    selectedTreePath = path
    selectedTreeKind = 'file'
    if (!openFilePaths.includes(path)) {
      openFilePaths = [...openFilePaths, path]
    }
  }

  function selectTreeNode(path: string, kind: SkillTreeKind) {
    selectedTreePath = path
    selectedTreeKind = kind
    if (kind === 'file') {
      selectFile(path)
    }
  }

  function closeTab(path: string) {
    const remaining = openFilePaths.filter((p) => p !== path)
    openFilePaths = remaining
    if (selectedFilePath === path) {
      selectedFilePath = remaining.at(-1) ?? null
      selectedTreePath = selectedFilePath
      selectedTreeKind = selectedFilePath ? 'file' : null
    }
  }

  function handleContentChange(path: string, value: string) {
    draftFiles = draftFiles.map((file) =>
      file.path === path ? updateDraftTextFileContent(file, value) : file,
    )
  }

  function replaceOpenPathPrefix(previousPath: string, nextPath: string) {
    openFilePaths = openFilePaths.map((path) =>
      path === previousPath || path.startsWith(`${previousPath}/`)
        ? `${nextPath}${path.slice(previousPath.length)}`
        : path,
    )
    if (
      selectedFilePath &&
      (selectedFilePath === previousPath || selectedFilePath.startsWith(`${previousPath}/`))
    ) {
      selectedFilePath = `${nextPath}${selectedFilePath.slice(previousPath.length)}`
    }
    if (
      selectedTreePath &&
      (selectedTreePath === previousPath || selectedTreePath.startsWith(`${previousPath}/`))
    ) {
      selectedTreePath = `${nextPath}${selectedTreePath.slice(previousPath.length)}`
    }
  }

  function removeOpenPathsUnder(targetPath: string) {
    openFilePaths = openFilePaths.filter(
      (path) => path !== targetPath && !path.startsWith(`${targetPath}/`),
    )
    if (
      selectedFilePath &&
      (selectedFilePath === targetPath || selectedFilePath.startsWith(`${targetPath}/`))
    ) {
      selectedFilePath = openFilePaths.at(-1) ?? null
    }
    if (
      selectedTreePath &&
      (selectedTreePath === targetPath || selectedTreePath.startsWith(`${targetPath}/`))
    ) {
      selectedTreePath = selectedFilePath
      selectedTreeKind = selectedFilePath ? 'file' : null
    }
  }

  function currentEntrypointContent() {
    return draftFiles.find((file) => file.path === 'SKILL.md')?.content ?? ''
  }

  function buildBundleRequestFiles() {
    return draftFiles.map((file) => ({
      path: file.path,
      content_base64: file.content_base64 ?? encodeUTF8Base64(file.content ?? ''),
      media_type: file.media_type,
      is_executable: file.is_executable,
    }))
  }

  // Inline creation: set pending state so tree renders inline input
  function handleCreateFile() {
    const parentPath =
      selectedTreeKind === 'directory'
        ? (selectedTreePath ?? '')
        : selectedTreePath
          ? selectedTreePath.includes('/')
            ? selectedTreePath.split('/').slice(0, -1).join('/')
            : ''
          : ''
    pendingCreate = { kind: 'file', parentPath }
  }

  function handleCreateFolder() {
    const parentPath =
      selectedTreeKind === 'directory'
        ? (selectedTreePath ?? '')
        : selectedTreePath
          ? selectedTreePath.includes('/')
            ? selectedTreePath.split('/').slice(0, -1).join('/')
            : ''
          : ''
    pendingCreate = { kind: 'folder', parentPath }
  }

  function handleCreateCommit(fullPath: string, kind: 'file' | 'folder') {
    pendingCreate = null
    try {
      if (kind === 'file') {
        draftFiles = addDraftTextFile(draftFiles, emptyDirectoryPaths, fullPath)
        const nextFile = draftFiles.at(-1)
        if (!nextFile) return
        selectFile(nextFile.path)
      } else {
        emptyDirectoryPaths = addEmptyDirectory(emptyDirectoryPaths, draftFiles, fullPath)
        selectedTreePath = normalizeSkillBundlePath(fullPath)
        selectedTreeKind = 'directory'
      }
    } catch (err) {
      toastStore.error(err instanceof Error ? err.message : `Failed to create ${kind}.`)
    }
  }

  function handleCreateCancel() {
    pendingCreate = null
  }

  function handleRenameNode(oldPath: string, newPath: string, kind: SkillTreeKind) {
    try {
      const normalizedNextPath = normalizeSkillBundlePath(newPath)
      if (kind === 'file') {
        draftFiles = renameFilePath(draftFiles, oldPath, normalizedNextPath)
        replaceOpenPathPrefix(oldPath, normalizedNextPath)
      } else {
        const renamed = renameDirectoryPath(
          draftFiles,
          emptyDirectoryPaths,
          oldPath,
          normalizedNextPath,
        )
        draftFiles = renamed.files
        emptyDirectoryPaths = renamed.emptyDirectoryPaths
        replaceOpenPathPrefix(oldPath, normalizedNextPath)
      }
      selectedTreePath = normalizedNextPath
      selectedTreeKind = kind
      if (kind === 'file') {
        selectedFilePath = normalizedNextPath
      }
    } catch (err) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to rename.')
    }
  }

  function handleDeleteNode(path: string, kind: SkillTreeKind) {
    const label = kind === 'directory' ? 'folder' : 'file'
    if (!window.confirm(`Delete ${label} "${path}"?`)) return

    try {
      if (kind === 'file') {
        draftFiles = deleteFilePath(draftFiles, path)
      } else {
        const deleted = deleteDirectoryPath(draftFiles, emptyDirectoryPaths, path)
        draftFiles = deleted.files
        emptyDirectoryPaths = deleted.emptyDirectoryPaths
      }
      removeOpenPathsUnder(path)
    } catch (err) {
      toastStore.error(err instanceof Error ? err.message : `Failed to delete ${label}.`)
    }
  }

  async function handleSave() {
    if (!skill) return

    const entrypointContent = currentEntrypointContent()

    if (!entrypointContent.trim()) {
      toastStore.error('Skill content is required.')
      return
    }
    if (!hasDirtyChanges) {
      if (emptyDraftDirectories.length > 0) {
        toastStore.warning('Empty folders are not persisted until they contain at least one file.')
      } else {
        toastStore.warning('No changes to save.')
      }
      return
    }
    if (emptyDraftDirectories.length > 0) {
      toastStore.warning('Empty folders are not persisted until they contain at least one file.')
    }

    busy = true
    try {
      await updateSkill(skill.id, {
        description: editDescription.trim(),
        content: entrypointContent,
        files: buildBundleRequestFiles(),
      })

      const loaded = await loadSkillEditorData(skill.id, appStore.currentProject?.id)

      skill = loaded.skill
      files = loaded.files
      draftFiles = loaded.files.map(cloneSkillFile)
      history = loaded.history
      workflows = loaded.workflows
      emptyDirectoryPaths = []
      editDescription = loaded.skill.description

      // Preserve open tabs where possible
      const validPaths = new Set(loaded.files.map((f) => f.path))
      openFilePaths = openFilePaths.filter((p) => validPaths.has(p))
      if (selectedFilePath && !validPaths.has(selectedFilePath)) {
        selectedFilePath = openFilePaths.at(-1) ?? null
        selectedTreePath = selectedFilePath
        selectedTreeKind = selectedFilePath ? 'file' : null
      }

      toastStore.success(`Saved ${skill.name}.`)
    } catch (err) {
      toastStore.error(err instanceof ApiError ? err.detail : 'Failed to save skill.')
    } finally {
      busy = false
    }
  }

  async function handleToggleEnabled() {
    if (!skill) return
    busy = true
    try {
      if (skill.is_enabled) {
        const res = await disableSkill(skill.id)
        skill = res.skill
      } else {
        const res = await enableSkill(skill.id)
        skill = res.skill
      }
      toastStore.success(`${skill.is_enabled ? 'Enabled' : 'Disabled'} ${skill.name}.`)
    } catch (err) {
      toastStore.error(err instanceof ApiError ? err.detail : 'Failed to update skill state.')
    } finally {
      busy = false
    }
  }

  async function handleDelete() {
    if (!skill) return
    const confirmed = window.confirm(`Delete "${skill.name}" and remove it from all workflows?`)
    if (!confirmed) return

    busy = true
    try {
      await deleteSkill(skill.id)
      toastStore.success(`Deleted ${skill.name}.`)
      navigateBack()
    } catch (err) {
      toastStore.error(err instanceof ApiError ? err.detail : 'Failed to delete skill.')
    } finally {
      busy = false
    }
  }

  async function handleWorkflowBinding(workflowId: string, shouldBind: boolean) {
    if (!skill) return
    busy = true
    try {
      if (shouldBind) {
        const res = await bindSkill(skill.id, [workflowId])
        skill = res.skill
      } else {
        const res = await unbindSkill(skill.id, [workflowId])
        skill = res.skill
      }
      const workflowName = workflows.find((w) => w.id === workflowId)?.name ?? 'workflow'
      toastStore.success(
        `${shouldBind ? 'Bound' : 'Unbound'} ${skill.name} ${shouldBind ? 'to' : 'from'} ${workflowName}.`,
      )
    } catch (err) {
      toastStore.error(err instanceof ApiError ? err.detail : 'Failed to update skill binding.')
    } finally {
      busy = false
    }
  }
</script>

<svelte:window onkeydown={handleKeydown} onbeforeunload={handleBeforeUnload} />

{#if loading}
  <div class="text-muted-foreground flex h-full items-center justify-center text-sm">
    Loading skill…
  </div>
{:else if !skill}
  <div class="text-muted-foreground flex h-full items-center justify-center text-sm">
    Skill not found.
  </div>
{:else}
  <div class="flex h-full flex-col" data-testid="skill-editor-page">
    <SkillEditorHeader
      {skill}
      {busy}
      {hasDirtyChanges}
      {metadataOpen}
      onNavigateBack={navigateBack}
      onSave={() => void handleSave()}
      onToggleEnabled={() => void handleToggleEnabled()}
      onDelete={() => void handleDelete()}
      onToggleMetadata={() => (metadataOpen = !metadataOpen)}
    />

    <SkillEditorWorkspace
      files={draftFiles}
      {emptyDirectoryPaths}
      selectedPath={selectedFilePath}
      {selectedTreePath}
      {openFiles}
      {dirtyPaths}
      {selectedFile}
      {activeContent}
      {skill}
      {workflows}
      {history}
      {busy}
      {metadataOpen}
      {pendingCreate}
      bind:editDescription
      onSelectTreeNode={selectTreeNode}
      onCreateFile={handleCreateFile}
      onCreateFolder={handleCreateFolder}
      onCreateCommit={handleCreateCommit}
      onCreateCancel={handleCreateCancel}
      onRenameNode={handleRenameNode}
      onDeleteNode={handleDeleteNode}
      onSelectTab={(path) => selectFile(path)}
      onCloseTab={closeTab}
      onContentChange={handleContentChange}
      onToggleBinding={handleWorkflowBinding}
    />

    <SkillEditorStatusBar
      {fileCount}
      totalSizeLabel={formatBytes(totalSize)}
      {selectedFile}
      dirtyCount={dirtyPaths.size}
    />
  </div>
{/if}
