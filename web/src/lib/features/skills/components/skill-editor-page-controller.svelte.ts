import { goto } from '$app/navigation'
import type { SkillFile, Skill, Workflow } from '$lib/api/contracts'
import { appStore } from '$lib/stores/app.svelte'
import { projectPath } from '$lib/stores/app-context'
import { computeDirtyPaths, listEmptyDirectories, type SkillTreeKind } from './skill-bundle-editor'
import { createSkillEditorPageControllerApi } from './skill-editor-page-controller-api'
import {
  createSkillEditorLoadEffect,
  createSkillEditorProjectFocusEffect,
} from './skill-editor-page-controller-effects'
import {
  handleSkillEditorBeforeUnload,
  handleSkillEditorKeydown,
  registerSkillEditorNavigationGuard,
} from './skill-editor-page-controller-navigation'
import {
  handleDeleteSkill,
  handleToggleSkillEnabled,
  handleWorkflowBindingMutation,
} from './skill-editor-page-controller-mutations'
import { createSkillEditorPageLayoutController } from './skill-editor-page-controller-layout'
import { createSkillEditorPageControllerState } from './skill-editor-page-controller-state'
import {
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
import type { SkillEditorHistoryEntry } from './skill-editor-page.helpers'

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
  const selectedFile = $derived(
    selectedFilePath ? (draftFiles.find((file) => file.path === selectedFilePath) ?? null) : null,
  )
  const openFiles = $derived(
    openFilePaths
      .map((path) => draftFiles.find((file) => file.path === path))
      .filter((file): file is SkillFile => file !== undefined),
  )
  const activeContent = $derived(selectedFile?.content ?? '')
  const selectedFileIsText = $derived(selectedFile?.encoding === 'utf8')
  const fileCount = $derived(draftFiles.length)
  const totalSize = $derived(draftFiles.reduce((sum, file) => sum + file.size_bytes, 0))
  const providers = $derived(appStore.providers ?? [])
  const layout = createSkillEditorPageLayoutController({
    getOpenFilePaths: () => openFilePaths,
    setOpenFilePaths: (value) => (openFilePaths = value),
    setSelectedFilePath: (value) => (selectedFilePath = value),
    setSelectedTreePath: (value) => (selectedTreePath = value),
    setSelectedTreeKind: (value) => (selectedTreeKind = value),
    setDragging: (value) => (dragging = value),
    getDragging: () => dragging,
    setDragStartX: (value) => (dragStartX = value),
    getDragStartX: () => dragStartX,
    setDragStartWidth: (value) => (dragStartWidth = value),
    getDragStartWidth: () => dragStartWidth,
    getAssistantWidth: () => assistantWidth,
    setAssistantWidth: (value) => (assistantWidth = value),
    minWidth: MIN_SIDEBAR_WIDTH,
    maxWidth: MAX_SIDEBAR_WIDTH,
  })

  const state: SkillEditorPageControllerActionsState = createSkillEditorPageControllerState({
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
    selectFile: layout.selectFile,
  })

  $effect(() => {
    return createSkillEditorLoadEffect({
      getSkillId: input.getSkillId,
      getProjectId: () => appStore.currentProject?.id,
      setLoading: (value) => (loading = value),
      state,
    })
  })

  $effect(() => {
    return createSkillEditorProjectFocusEffect({
      projectId: appStore.currentProject?.id,
      loading,
      skill,
      selectedFilePath,
      hasDirtyChanges,
    })
  })

  registerSkillEditorNavigationGuard({
    getHasDirtyChanges: () => hasDirtyChanges,
    getBusy: () => busy,
    handleSave,
  })

  function handleKeydown(event: KeyboardEvent) {
    handleSkillEditorKeydown(
      {
        getHasDirtyChanges: () => hasDirtyChanges,
        getBusy: () => busy,
        handleSave,
      },
      event,
    )
  }

  function handleBeforeUnload(event: BeforeUnloadEvent) {
    handleSkillEditorBeforeUnload(hasDirtyChanges, event)
  }

  function navigateBack() {
    const orgId = appStore.currentOrg?.id
    const projectId = appStore.currentProject?.id
    if (orgId && projectId) void goto(projectPath(orgId, projectId, 'skills'))
  }

  function selectFile(path: string) {
    layout.selectFile(path)
  }

  function selectTreeNode(path: string, kind: SkillTreeKind) {
    layout.selectTreeNode(path, kind)
  }

  function handleDragStart(event: PointerEvent) {
    layout.handleDragStart(event)
  }

  function handleDragMove(event: PointerEvent) {
    layout.handleDragMove(event)
  }

  function handleDragEnd() {
    layout.handleDragEnd()
  }

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
    await handleToggleSkillEnabled({
      getSkill: () => skill,
      setSkill: (value) => (skill = value),
      getWorkflows: () => workflows,
      setBusy: (value) => (busy = value),
      navigateBack,
    })
  }

  async function handleDelete() {
    await handleDeleteSkill({
      getSkill: () => skill,
      setSkill: (value) => (skill = value),
      getWorkflows: () => workflows,
      setBusy: (value) => (busy = value),
      navigateBack,
    })
  }

  async function handleWorkflowBinding(workflowId: string, shouldBind: boolean) {
    await handleWorkflowBindingMutation(
      {
        getSkill: () => skill,
        setSkill: (value) => (skill = value),
        getWorkflows: () => workflows,
        setBusy: (value) => (busy = value),
        navigateBack,
      },
      workflowId,
      shouldBind,
    )
  }

  return createSkillEditorPageControllerApi({
    getSkill: () => skill,
    getFiles: () => files,
    getDraftFiles: () => draftFiles,
    getEmptyDirectoryPaths: () => emptyDirectoryPaths,
    getHistory: () => history,
    getWorkflows: () => workflows,
    getLoading: () => loading,
    getBusy: () => busy,
    getEditDescription: () => editDescription,
    getShowAssistant: () => showAssistant,
    setShowAssistant: (value) => (showAssistant = value),
    getAssistantWidth: () => assistantWidth,
    getDragging: () => dragging,
    getSelectedFilePath: () => selectedFilePath,
    getSelectedTreePath: () => selectedTreePath,
    getOpenFiles: () => openFiles,
    getDirtyPaths: () => dirtyPaths,
    getSelectedFile: () => selectedFile,
    getActiveContent: () => activeContent,
    getPendingCreate: () => pendingCreate,
    getHasDirtyChanges: () => hasDirtyChanges,
    getSelectedFileIsText: () => selectedFileIsText,
    getFileCount: () => fileCount,
    getTotalSize: () => totalSize,
    getProviders: () => providers,
    setEditDescription,
    handleKeydown,
    handleBeforeUnload,
    navigateBack,
    selectFile,
    selectTreeNode,
    handleDragStart,
    handleDragMove,
    handleDragEnd,
    closeTab: (path) => closeEditorTab(state, path),
    handleContentChange: (path, value) => updateEditorContent(state, path, value),
    handleApplyAssistantSuggestion: (suggestedFiles: SkillFile[], focusPath?: string) =>
      applyAssistantSuggestion(state, suggestedFiles, focusPath),
    handleCreateFile: () => createFileDraft(state),
    handleCreateFolder: () => createFolderDraft(state),
    handleCreateCommit: (fullPath, kind) => commitPendingCreate(state, fullPath, kind),
    handleCreateCancel: () => cancelPendingCreate(state),
    handleRenameNode: (oldPath, newPath, kind) => renameTreeNode(state, oldPath, newPath, kind),
    handleDeleteNode: (path, kind) => deleteTreeNode(state, path, kind),
    handleSave,
    handleToggleEnabled,
    handleDelete,
    handleWorkflowBinding,
  })
}
