import type { Skill, SkillFile, Workflow } from '$lib/api/contracts'
import type { SkillTreeKind } from './skill-bundle-editor'
import type {
  SkillEditorPageControllerActionsState,
  SkillEditorPendingCreate,
} from './skill-editor-page-controller-actions'
import type { SkillEditorHistoryEntry } from './skill-editor-page.helpers'

type SkillEditorPageControllerStateInput = {
  getSkill: () => Skill | null
  setSkill: (value: Skill | null) => void
  setFiles: (value: SkillFile[]) => void
  getDraftFiles: () => SkillFile[]
  setDraftFiles: (value: SkillFile[]) => void
  getEmptyDirectoryPaths: () => string[]
  setEmptyDirectoryPaths: (value: string[]) => void
  getHistory: () => SkillEditorHistoryEntry[]
  setHistory: (value: SkillEditorHistoryEntry[]) => void
  getWorkflows: () => Workflow[]
  setWorkflows: (value: Workflow[]) => void
  getEditDescription: () => string
  setEditDescription: (value: string) => void
  getOpenFilePaths: () => string[]
  setOpenFilePaths: (value: string[]) => void
  getSelectedFilePath: () => string | null
  setSelectedFilePath: (value: string | null) => void
  getSelectedTreePath: () => string | null
  setSelectedTreePath: (value: string | null) => void
  setSelectedTreeKind: (value: SkillTreeKind | null) => void
  getSelectedTreePathParent: () => string
  getSelectedTreeKind: () => SkillTreeKind | null
  getPendingCreate: () => SkillEditorPendingCreate
  setPendingCreate: (value: SkillEditorPendingCreate) => void
  getHasDirtyChanges: () => boolean
  getEmptyDraftDirectories: () => string[]
  selectFile: (path: string) => void
}

export function createSkillEditorPageControllerState(
  input: SkillEditorPageControllerStateInput,
): SkillEditorPageControllerActionsState {
  return {
    getSkill: input.getSkill,
    setSkill: input.setSkill,
    setFiles: input.setFiles,
    getDraftFiles: input.getDraftFiles,
    setDraftFiles: input.setDraftFiles,
    getEmptyDirectoryPaths: input.getEmptyDirectoryPaths,
    setEmptyDirectoryPaths: input.setEmptyDirectoryPaths,
    getHistory: input.getHistory,
    setHistory: input.setHistory,
    getWorkflows: input.getWorkflows,
    setWorkflows: input.setWorkflows,
    getEditDescription: input.getEditDescription,
    setEditDescription: input.setEditDescription,
    getOpenFilePaths: input.getOpenFilePaths,
    setOpenFilePaths: input.setOpenFilePaths,
    getSelectedFilePath: input.getSelectedFilePath,
    setSelectedFilePath: input.setSelectedFilePath,
    getSelectedTreePath: input.getSelectedTreePath,
    setSelectedTreePath: input.setSelectedTreePath,
    setSelectedTreeKind: input.setSelectedTreeKind,
    getSelectedTreePathParent: input.getSelectedTreePathParent,
    getSelectedTreeKind: input.getSelectedTreeKind,
    getPendingCreate: input.getPendingCreate,
    setPendingCreate: input.setPendingCreate,
    getHasDirtyChanges: input.getHasDirtyChanges,
    getEmptyDraftDirectories: input.getEmptyDraftDirectories,
    selectFile: input.selectFile,
  }
}
