import type { AgentProvider, Skill, SkillFile, Workflow } from '$lib/api/contracts'
import type { SkillTreeKind } from './skill-bundle-editor'
import type { SkillEditorPendingCreate } from './skill-editor-page-controller-actions'
import type { SkillEditorHistoryEntry } from './skill-editor-page.helpers'

type SkillEditorPageControllerApiInput = {
  getSkill: () => Skill | null
  getFiles: () => SkillFile[]
  getDraftFiles: () => SkillFile[]
  getEmptyDirectoryPaths: () => string[]
  getHistory: () => SkillEditorHistoryEntry[]
  getWorkflows: () => Workflow[]
  getLoading: () => boolean
  getBusy: () => boolean
  getEditDescription: () => string
  getShowAssistant: () => boolean
  setShowAssistant: (value: boolean) => void
  getAssistantWidth: () => number
  getDragging: () => boolean
  getSelectedFilePath: () => string | null
  getSelectedTreePath: () => string | null
  getOpenFiles: () => SkillFile[]
  getDirtyPaths: () => Set<string>
  getSelectedFile: () => SkillFile | null
  getActiveContent: () => string
  getPendingCreate: () => SkillEditorPendingCreate
  getHasDirtyChanges: () => boolean
  getSelectedFileIsText: () => boolean
  getFileCount: () => number
  getTotalSize: () => number
  getProviders: () => AgentProvider[]
  setEditDescription: (value: string) => void
  handleKeydown: (event: KeyboardEvent) => void
  handleBeforeUnload: (event: BeforeUnloadEvent) => void
  navigateBack: () => void
  selectFile: (path: string) => void
  selectTreeNode: (path: string, kind: SkillTreeKind) => void
  handleDragStart: (event: PointerEvent) => void
  handleDragMove: (event: PointerEvent) => void
  handleDragEnd: () => void
  handleApplyAssistantSuggestion: (suggestedFiles: SkillFile[], focusPath?: string) => void
  handleSave: () => Promise<void>
  handleToggleEnabled: () => Promise<void>
  handleDelete: () => Promise<void>
  handleWorkflowBinding: (workflowId: string, shouldBind: boolean) => Promise<void>
  closeTab: (path: string) => void
  handleContentChange: (path: string, value: string) => void
  handleCreateFile: () => void
  handleCreateFolder: () => void
  handleCreateCommit: (fullPath: string, kind: 'file' | 'folder') => void
  handleCreateCancel: () => void
  handleRenameNode: (oldPath: string, newPath: string, kind: SkillTreeKind) => void
  handleDeleteNode: (path: string, kind: SkillTreeKind) => void
}

export function createSkillEditorPageControllerApi(input: SkillEditorPageControllerApiInput) {
  return {
    get skill() {
      return input.getSkill()
    },
    get files() {
      return input.getFiles()
    },
    get draftFiles() {
      return input.getDraftFiles()
    },
    get emptyDirectoryPaths() {
      return input.getEmptyDirectoryPaths()
    },
    get history() {
      return input.getHistory()
    },
    get workflows() {
      return input.getWorkflows()
    },
    get loading() {
      return input.getLoading()
    },
    get busy() {
      return input.getBusy()
    },
    get editDescription() {
      return input.getEditDescription()
    },
    get showAssistant() {
      return input.getShowAssistant()
    },
    set showAssistant(value: boolean) {
      input.setShowAssistant(value)
    },
    get assistantWidth() {
      return input.getAssistantWidth()
    },
    get dragging() {
      return input.getDragging()
    },
    get selectedFilePath() {
      return input.getSelectedFilePath()
    },
    get selectedTreePath() {
      return input.getSelectedTreePath()
    },
    get openFiles() {
      return input.getOpenFiles()
    },
    get dirtyPaths() {
      return input.getDirtyPaths()
    },
    get selectedFile() {
      return input.getSelectedFile()
    },
    get activeContent() {
      return input.getActiveContent()
    },
    get pendingCreate() {
      return input.getPendingCreate()
    },
    get hasDirtyChanges() {
      return input.getHasDirtyChanges()
    },
    get selectedFileIsText() {
      return input.getSelectedFileIsText()
    },
    get fileCount() {
      return input.getFileCount()
    },
    get totalSize() {
      return input.getTotalSize()
    },
    get providers() {
      return input.getProviders()
    },
    setEditDescription: input.setEditDescription,
    handleKeydown: input.handleKeydown,
    handleBeforeUnload: input.handleBeforeUnload,
    navigateBack: input.navigateBack,
    selectFile: input.selectFile,
    selectTreeNode: input.selectTreeNode,
    closeTab: input.closeTab,
    handleContentChange: input.handleContentChange,
    handleApplyAssistantSuggestion: input.handleApplyAssistantSuggestion,
    handleDragStart: input.handleDragStart,
    handleDragMove: input.handleDragMove,
    handleDragEnd: input.handleDragEnd,
    handleCreateFile: input.handleCreateFile,
    handleCreateFolder: input.handleCreateFolder,
    handleCreateCommit: input.handleCreateCommit,
    handleCreateCancel: input.handleCreateCancel,
    handleRenameNode: input.handleRenameNode,
    handleDeleteNode: input.handleDeleteNode,
    handleSave: input.handleSave,
    handleToggleEnabled: input.handleToggleEnabled,
    handleDelete: input.handleDelete,
    handleWorkflowBinding: input.handleWorkflowBinding,
  }
}
