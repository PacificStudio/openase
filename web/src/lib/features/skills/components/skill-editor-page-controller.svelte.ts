import { goto, beforeNavigate } from '$app/navigation'
import { PROJECT_AI_FOCUS_PRIORITY } from '$lib/features/chat'
import { ApiError } from '$lib/api/client'
import type { SkillFile, Skill, Workflow } from '$lib/api/contracts'
import { bindSkill, deleteSkill, disableSkill, enableSkill, unbindSkill } from '$lib/api/openase'
import { appStore } from '$lib/stores/app.svelte'
import { projectPath } from '$lib/stores/app-context'
import { toastStore } from '$lib/stores/toast.svelte'
import { computeDirtyPaths, listEmptyDirectories, type SkillTreeKind } from './skill-bundle-editor'
import {
  applyInitialSkillLoad,
  closeTab as closeEditorTab,
  handleApplyAssistantSuggestion as applyAssistantSuggestion,
  handleContentChange as updateEditorContent,
  handleCreateCancel as cancelPendingCreate,
  handleCreateCommit as commitPendingCreate,
  handleCreateFile as createFileDraft,
  handleCreateFolder as createFolderDraft,
  handleDeleteNode as deleteTreeNode,
  handleRenameNode as renameTreeNode,
  handleSave as persistSkillDraft,
  type SkillEditorPageControllerActionsState,
  type SkillEditorPendingCreate,
} from './skill-editor-page-controller-actions'
import { loadSkillEditorData, type SkillEditorHistoryEntry } from './skill-editor-page.helpers'

const projectAIFocusOwner = 'skill-editor-page'
const MIN_SIDEBAR_WIDTH = 260
const MAX_SIDEBAR_WIDTH = 560

export function createSkillEditorPageController(input: { getSkillId: () => string }) {
  let skill = $state<Skill | null>(null)
  let files = $state<SkillFile[]>([])
  let draftFiles = $state<SkillFile[]>([])
  let emptyDirectoryPaths = $state<string[]>([])
  let history = $state<SkillEditorHistoryEntry[]>([])
  let workflows = $state<Workflow[]>([])
  let loading = $state(true)
  let busy = $state(false)
  let editDescription = $state('')
  let showAssistant = $state(false)
  let assistantWidth = $state(340)
  let dragging = $state(false)
  let dragStartX = $state(0)
  let dragStartWidth = $state(0)
  let selectedFilePath = $state<string | null>(null)
  let selectedTreePath = $state<string | null>(null)
  let selectedTreeKind = $state<SkillTreeKind | null>(null)
  let openFilePaths = $state<string[]>([])
  let pendingCreate = $state<SkillEditorPendingCreate>(null)

  const dirtyPaths = $derived(computeDirtyPaths(files, draftFiles))
  const descriptionDirty = $derived(skill ? editDescription.trim() !== skill.description : false)
  const hasDirtyChanges = $derived(dirtyPaths.size > 0 || descriptionDirty)
  const emptyDraftDirectories = $derived(listEmptyDirectories(draftFiles, emptyDirectoryPaths))
  const selectedFile = $derived(selectedFilePath ? (draftFiles.find((file) => file.path === selectedFilePath) ?? null) : null)
  const openFiles = $derived(
    openFilePaths.map((path) => draftFiles.find((file) => file.path === path)).filter((file): file is SkillFile => file !== undefined),
  )
  const activeContent = $derived(selectedFile?.content ?? '')
  const selectedFileIsText = $derived(selectedFile?.encoding === 'utf8')
  const fileCount = $derived(draftFiles.length)
  const totalSize = $derived(draftFiles.reduce((sum, file) => sum + file.size_bytes, 0))
  const providers = $derived(appStore.providers ?? [])

  const state: SkillEditorPageControllerActionsState = {
    getSkill: () => skill,
    setSkill: (value) => (skill = value),
    setFiles: (value) => (files = value),
    getDraftFiles: () => draftFiles,
    setDraftFiles: (value) => (draftFiles = value),
    getEmptyDirectoryPaths: () => emptyDirectoryPaths,
    setEmptyDirectoryPaths: (value) => (emptyDirectoryPaths = value),
    getHistory: () => history,
    setHistory: (value) => (history = value),
    getWorkflows: () => workflows,
    setWorkflows: (value) => (workflows = value),
    getEditDescription: () => editDescription,
    setEditDescription: (value) => (editDescription = value),
    getOpenFilePaths: () => openFilePaths,
    setOpenFilePaths: (value) => (openFilePaths = value),
    getSelectedFilePath: () => selectedFilePath,
    setSelectedFilePath: (value) => (selectedFilePath = value),
    getSelectedTreePath: () => selectedTreePath,
    setSelectedTreePath: (value) => (selectedTreePath = value),
    setSelectedTreeKind: (value) => (selectedTreeKind = value),
    getSelectedTreePathParent: () =>
      selectedTreeKind === 'directory'
        ? (selectedTreePath ?? '')
        : selectedTreePath
          ? selectedTreePath.split('/').slice(0, -1).join('/')
          : '',
    getSelectedTreeKind: () => selectedTreeKind,
    getPendingCreate: () => pendingCreate,
    setPendingCreate: (value) => (pendingCreate = value),
    getHasDirtyChanges: () => hasDirtyChanges,
    getEmptyDraftDirectories: () => emptyDraftDirectories,
    selectFile,
  }

  $effect(() => {
    const skillId = input.getSkillId()
    if (!skillId) return
    let cancelled = false
    loading = true
    void (async () => {
      try {
        const loaded = await loadSkillEditorData(skillId, appStore.currentProject?.id)
        if (!cancelled) applyInitialSkillLoad(state, loaded)
      } catch (err) {
        if (!cancelled) toastStore.error(err instanceof ApiError ? err.detail : 'Failed to load skill.')
      } finally {
        if (!cancelled) loading = false
      }
    })()
    return () => { cancelled = true }
  })

  $effect(() => {
    const projectId = appStore.currentProject?.id
    if (!projectId || loading || !skill) {
      appStore.clearProjectAssistantFocus(projectAIFocusOwner)
      return
    }
    appStore.setProjectAssistantFocus(
      projectAIFocusOwner,
      {
        kind: 'skill',
        projectId,
        skillId: skill.id,
        skillName: skill.name,
        selectedFilePath: selectedFilePath ?? 'SKILL.md',
        boundWorkflowNames: skill.bound_workflows.map((workflow) => workflow.name),
        hasDirtyDraft: hasDirtyChanges,
      },
      PROJECT_AI_FOCUS_PRIORITY.workspace,
    )
    return () => { appStore.clearProjectAssistantFocus(projectAIFocusOwner) }
  })

  beforeNavigate(({ cancel }) => {
    if (hasDirtyChanges && !window.confirm('You have unsaved changes. Leave without saving?')) {
      cancel()
    }
  })

  function handleKeydown(event: KeyboardEvent) {
    if ((event.metaKey || event.ctrlKey) && event.key === 's') {
      event.preventDefault()
      if (!busy && hasDirtyChanges) void handleSave()
    }
  }

  function handleBeforeUnload(event: BeforeUnloadEvent) {
    if (hasDirtyChanges) event.preventDefault()
  }

  function navigateBack() {
    const orgId = appStore.currentOrg?.id
    const projectId = appStore.currentProject?.id
    if (orgId && projectId) void goto(projectPath(orgId, projectId, 'skills'))
  }

  function selectFile(path: string) {
    selectedFilePath = path
    selectedTreePath = path
    selectedTreeKind = 'file'
    if (!openFilePaths.includes(path)) openFilePaths = [...openFilePaths, path]
  }

  function selectTreeNode(path: string, kind: SkillTreeKind) {
    selectedTreePath = path
    selectedTreeKind = kind
    if (kind === 'file') selectFile(path)
  }

  function handleDragStart(event: PointerEvent) {
    dragging = true
    dragStartX = event.clientX
    dragStartWidth = assistantWidth
    ;(event.target as HTMLElement).setPointerCapture(event.pointerId)
  }

  function handleDragMove(event: PointerEvent) {
    if (!dragging) return
    assistantWidth = Math.min(MAX_SIDEBAR_WIDTH, Math.max(MIN_SIDEBAR_WIDTH, dragStartWidth + dragStartX - event.clientX))
  }

  function handleDragEnd() { dragging = false }

  function setEditDescription(value: string) {
    editDescription = value
  }

  async function handleSave() {
    if (!skill) return
    busy = true
    try {
      await persistSkillDraft(state, input.getSkillId(), appStore.currentProject?.id)
    } finally {
      busy = false
    }
  }

  async function handleToggleEnabled() {
    if (!skill) return
    busy = true
    try {
      skill = skill.is_enabled ? (await disableSkill(skill.id)).skill : (await enableSkill(skill.id)).skill
      toastStore.success(`${skill.is_enabled ? 'Enabled' : 'Disabled'} ${skill.name}.`)
    } catch (err) {
      toastStore.error(err instanceof ApiError ? err.detail : 'Failed to update skill state.')
    } finally {
      busy = false
    }
  }

  async function handleDelete() {
    if (!skill || !window.confirm(`Delete "${skill.name}" and remove it from all workflows?`)) return
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
      skill = shouldBind ? (await bindSkill(skill.id, [workflowId])).skill : (await unbindSkill(skill.id, [workflowId])).skill
      const workflowName = workflows.find((workflow) => workflow.id === workflowId)?.name ?? 'workflow'
      toastStore.success(`${shouldBind ? 'Bound' : 'Unbound'} ${skill.name} ${shouldBind ? 'to' : 'from'} ${workflowName}.`)
    } catch (err) {
      toastStore.error(err instanceof ApiError ? err.detail : 'Failed to update skill binding.')
    } finally {
      busy = false
    }
  }

  return {
    get skill() { return skill },
    get files() { return files },
    get draftFiles() { return draftFiles },
    get emptyDirectoryPaths() { return emptyDirectoryPaths },
    get history() { return history },
    get workflows() { return workflows },
    get loading() { return loading },
    get busy() { return busy },
    get editDescription() { return editDescription },
    get showAssistant() { return showAssistant },
    set showAssistant(value: boolean) { showAssistant = value },
    get assistantWidth() { return assistantWidth },
    get dragging() { return dragging },
    get selectedFilePath() { return selectedFilePath },
    get selectedTreePath() { return selectedTreePath },
    get openFiles() { return openFiles },
    get dirtyPaths() { return dirtyPaths },
    get selectedFile() { return selectedFile },
    get activeContent() { return activeContent },
    get pendingCreate() { return pendingCreate },
    get hasDirtyChanges() { return hasDirtyChanges },
    get selectedFileIsText() { return selectedFileIsText },
    get fileCount() { return fileCount },
    get totalSize() { return totalSize },
    get providers() { return providers },
    setEditDescription,
    handleKeydown,
    handleBeforeUnload,
    navigateBack,
    selectFile,
    selectTreeNode,
    closeTab: (path: string) => closeEditorTab(state, path),
    handleContentChange: (path: string, value: string) => updateEditorContent(state, path, value),
    handleApplyAssistantSuggestion: (suggestedFiles: SkillFile[], focusPath?: string) =>
      applyAssistantSuggestion(state, suggestedFiles, focusPath),
    handleDragStart,
    handleDragMove,
    handleDragEnd,
    handleCreateFile: () => createFileDraft(state),
    handleCreateFolder: () => createFolderDraft(state),
    handleCreateCommit: (fullPath: string, kind: 'file' | 'folder') =>
      commitPendingCreate(state, fullPath, kind),
    handleCreateCancel: () => cancelPendingCreate(state),
    handleRenameNode: (oldPath: string, newPath: string, kind: SkillTreeKind) =>
      renameTreeNode(state, oldPath, newPath, kind),
    handleDeleteNode: (path: string, kind: SkillTreeKind) => deleteTreeNode(state, path, kind),
    handleSave,
    handleToggleEnabled,
    handleDelete,
    handleWorkflowBinding,
  }
}
