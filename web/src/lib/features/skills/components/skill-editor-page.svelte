<script lang="ts">
  import { goto } from '$app/navigation'
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
    defaultChildDirectoryPath,
    defaultChildFilePath,
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
  let editing = $state(false)
  let editDescription = $state('')
  let metadataOpen = $state(true)

  let selectedFilePath = $state<string | null>(null)
  let selectedTreePath = $state<string | null>(null)
  let selectedTreeKind = $state<SkillTreeKind | null>(null)
  let openFilePaths = $state<string[]>([])

  const displayedFiles = $derived(editing ? draftFiles : files)
  const dirtyPaths = $derived(editing ? computeDirtyPaths(files, draftFiles) : new Set<string>())
  const emptyDraftDirectories = $derived(
    editing ? listEmptyDirectories(draftFiles, emptyDirectoryPaths) : [],
  )

  const selectedFile = $derived(
    selectedFilePath ? (displayedFiles.find((f) => f.path === selectedFilePath) ?? null) : null,
  )

  const openFiles = $derived(
    openFilePaths
      .map((p) => displayedFiles.find((f) => f.path === p))
      .filter((f): f is SkillFile => f !== undefined),
  )

  const activeContent = $derived(selectedFile?.content ?? '')

  const fileCount = $derived(displayedFiles.length)
  const totalSize = $derived(displayedFiles.reduce((sum, f) => sum + f.size_bytes, 0))

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
        draftFiles = []
        emptyDirectoryPaths = []
        history = loaded.history
        workflows = loaded.workflows

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

  function startEditing() {
    if (!skill) return
    editDescription = skill.description
    draftFiles = files.map(cloneSkillFile)
    emptyDirectoryPaths = []
    editing = true
  }

  function cancelEditing() {
    editing = false
    draftFiles = []
    emptyDirectoryPaths = []
    selectedTreePath = selectedFilePath
    selectedTreeKind = selectedFilePath ? 'file' : null
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
    return displayedFiles.find((file) => file.path === 'SKILL.md')?.content ?? ''
  }

  function buildBundleRequestFiles() {
    return draftFiles.map((file) => ({
      path: file.path,
      content_base64: file.content_base64 ?? encodeUTF8Base64(file.content ?? ''),
      media_type: file.media_type,
      is_executable: file.is_executable,
    }))
  }

  function promptValue(message: string, initialValue: string): string | null {
    const nextValue = window.prompt(message, initialValue)
    if (nextValue === null) return null
    if (!nextValue.trim()) {
      toastStore.error('Path is required.')
      return null
    }
    return nextValue
  }

  function handleCreateFile() {
    const requestedPath = promptValue(
      'New file path',
      defaultChildFilePath(selectedTreePath, selectedTreeKind),
    )
    if (!requestedPath) return
    try {
      draftFiles = addDraftTextFile(draftFiles, emptyDirectoryPaths, requestedPath)
      const nextFile = draftFiles.at(-1)
      if (!nextFile) return
      selectFile(nextFile.path)
    } catch (err) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to create file.')
    }
  }

  function handleCreateFolder() {
    const requestedPath = promptValue(
      'New folder path',
      defaultChildDirectoryPath(selectedTreePath, selectedTreeKind),
    )
    if (!requestedPath) return
    try {
      emptyDirectoryPaths = addEmptyDirectory(emptyDirectoryPaths, draftFiles, requestedPath)
      selectedTreePath = normalizeSkillBundlePath(requestedPath)
      selectedTreeKind = 'directory'
    } catch (err) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to create folder.')
    }
  }

  function handleRenameSelection() {
    if (!selectedTreePath || !selectedTreeKind) return
    const requestedPath = promptValue(
      selectedTreeKind === 'directory' ? 'Rename folder to' : 'Rename file to',
      selectedTreePath,
    )
    if (!requestedPath) return
    try {
      const normalizedNextPath = normalizeSkillBundlePath(requestedPath)
      if (selectedTreeKind === 'file') {
        draftFiles = renameFilePath(draftFiles, selectedTreePath, normalizedNextPath)
        replaceOpenPathPrefix(selectedTreePath, normalizedNextPath)
      } else {
        const renamed = renameDirectoryPath(
          draftFiles,
          emptyDirectoryPaths,
          selectedTreePath,
          normalizedNextPath,
        )
        draftFiles = renamed.files
        emptyDirectoryPaths = renamed.emptyDirectoryPaths
        replaceOpenPathPrefix(selectedTreePath, normalizedNextPath)
      }
      selectedTreePath = normalizedNextPath
    } catch (err) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to rename selection.')
    }
  }

  function handleDeleteSelection() {
    if (!selectedTreePath || !selectedTreeKind) return
    const label = selectedTreeKind === 'directory' ? 'folder' : 'file'
    if (!window.confirm(`Delete ${label} "${selectedTreePath}"?`)) return

    try {
      if (selectedTreeKind === 'file') {
        draftFiles = deleteFilePath(draftFiles, selectedTreePath)
      } else {
        const deleted = deleteDirectoryPath(draftFiles, emptyDirectoryPaths, selectedTreePath)
        draftFiles = deleted.files
        emptyDirectoryPaths = deleted.emptyDirectoryPaths
      }
      removeOpenPathsUnder(selectedTreePath)
    } catch (err) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to delete selection.')
    }
  }

  async function handleSave() {
    if (!skill) return

    const entrypointContent = currentEntrypointContent()

    if (!entrypointContent.trim()) {
      toastStore.error('Skill content is required.')
      return
    }
    if (dirtyPaths.size === 0 && editDescription.trim() === skill.description) {
      if (emptyDraftDirectories.length > 0) {
        toastStore.warning('Empty folders are not persisted until they contain at least one file.')
      } else {
        toastStore.warning('No changes to publish.')
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
      history = loaded.history
      workflows = loaded.workflows
      editing = false
      draftFiles = []
      emptyDirectoryPaths = []
      const selection = selectInitialSkillFiles(loaded.files)
      selectedFilePath = selection.selectedFilePath
      selectedTreePath = selection.selectedFilePath
      selectedTreeKind = selection.selectedFilePath ? 'file' : null
      openFilePaths = selection.openFilePaths
      toastStore.success(`Updated ${skill.name}.`)
    } catch (err) {
      toastStore.error(err instanceof ApiError ? err.detail : 'Failed to update skill.')
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
      {editing}
      {busy}
      {metadataOpen}
      onNavigateBack={navigateBack}
      onCancelEditing={cancelEditing}
      onSave={() => void handleSave()}
      onStartEditing={startEditing}
      onToggleEnabled={() => void handleToggleEnabled()}
      onDelete={() => void handleDelete()}
      onToggleMetadata={() => (metadataOpen = !metadataOpen)}
    />

    <SkillEditorWorkspace
      files={displayedFiles}
      {emptyDirectoryPaths}
      selectedPath={selectedFilePath}
      {selectedTreePath}
      {selectedTreeKind}
      {openFiles}
      {dirtyPaths}
      {selectedFile}
      {activeContent}
      {skill}
      {workflows}
      {history}
      {editing}
      {busy}
      {metadataOpen}
      bind:editDescription
      onSelectTreeNode={selectTreeNode}
      onCreateFile={handleCreateFile}
      onCreateFolder={handleCreateFolder}
      onRenameSelection={handleRenameSelection}
      onDeleteSelection={handleDeleteSelection}
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
      {editing}
    />
  </div>
{/if}
