import { updateSkill } from '$lib/api/openase'
import type { Skill, SkillFile, Workflow } from '$lib/api/contracts'
import { ApiError } from '$lib/api/client'
import { toastStore } from '$lib/stores/toast.svelte'
import {
  addDraftTextFile,
  addEmptyDirectory,
  cloneSkillFile,
  deleteDirectoryPath,
  deleteFilePath,
  normalizeSkillBundlePath,
  renameDirectoryPath,
  renameFilePath,
  type SkillTreeKind,
  updateDraftTextFileContent,
} from './skill-bundle-editor'
import {
  buildBundleRequestFiles,
  currentEntrypointContent,
  removeOpenPathsUnder,
  replaceOpenPathPrefix,
} from './skill-editor-page-controller-action-helpers'
import {
  loadSkillEditorData,
  selectInitialSkillFiles,
  type SkillEditorHistoryEntry,
} from './skill-editor-page.helpers'

export type SkillEditorPendingCreate = { kind: 'file' | 'folder'; parentPath: string } | null

export type SkillEditorPageControllerActionsState = {
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

export function applyInitialSkillLoad(
  state: SkillEditorPageControllerActionsState,
  loaded: Awaited<ReturnType<typeof loadSkillEditorData>>,
) {
  state.setSkill(loaded.skill)
  state.setFiles(loaded.files)
  state.setDraftFiles(loaded.files.map(cloneSkillFile))
  state.setEmptyDirectoryPaths([])
  state.setHistory(loaded.history)
  state.setWorkflows(loaded.workflows)
  state.setEditDescription(loaded.skill.description)
  const selection = selectInitialSkillFiles(loaded.files)
  state.setSelectedFilePath(selection.selectedFilePath)
  state.setSelectedTreePath(selection.selectedFilePath)
  state.setSelectedTreeKind(selection.selectedFilePath ? 'file' : null)
  state.setOpenFilePaths(selection.openFilePaths)
}

export function applySavedSkillLoad(
  state: SkillEditorPageControllerActionsState,
  loaded: Awaited<ReturnType<typeof loadSkillEditorData>>,
) {
  state.setSkill(loaded.skill)
  state.setFiles(loaded.files)
  state.setDraftFiles(loaded.files.map(cloneSkillFile))
  state.setHistory(loaded.history)
  state.setWorkflows(loaded.workflows)
  state.setEmptyDirectoryPaths([])
  state.setEditDescription(loaded.skill.description)

  const validPaths = new Set(loaded.files.map((file) => file.path))
  state.setOpenFilePaths(state.getOpenFilePaths().filter((path) => validPaths.has(path)))
  const selectedFilePath = state.getSelectedFilePath()
  if (selectedFilePath && !validPaths.has(selectedFilePath)) {
    state.setSelectedFilePath(state.getOpenFilePaths().at(-1) ?? null)
    state.setSelectedTreePath(state.getSelectedFilePath())
    state.setSelectedTreeKind(state.getSelectedFilePath() ? 'file' : null)
  }
}

export function closeTab(state: SkillEditorPageControllerActionsState, path: string) {
  const remaining = state.getOpenFilePaths().filter((current) => current !== path)
  state.setOpenFilePaths(remaining)
  if (state.getSelectedFilePath() === path) {
    state.setSelectedFilePath(remaining.at(-1) ?? null)
    state.setSelectedTreePath(state.getSelectedFilePath())
    state.setSelectedTreeKind(state.getSelectedFilePath() ? 'file' : null)
  }
}

export function handleContentChange(
  state: SkillEditorPageControllerActionsState,
  path: string,
  value: string,
) {
  state.setDraftFiles(
    state
      .getDraftFiles()
      .map((file) => (file.path === path ? updateDraftTextFileContent(file, value) : file)),
  )
}

export function handleApplyAssistantSuggestion(
  state: SkillEditorPageControllerActionsState,
  suggestedFiles: SkillFile[],
  focusPath?: string,
) {
  if (suggestedFiles.length === 0) return
  state.setDraftFiles(suggestedFiles.map(cloneSkillFile))
  state.setEmptyDirectoryPaths([])
  const validPaths = new Set(suggestedFiles.map((file) => file.path))
  state.setOpenFilePaths(state.getOpenFilePaths().filter((path) => validPaths.has(path)))
  const nextFocusPath =
    focusPath ??
    suggestedFiles.find((file) => file.encoding === 'utf8')?.path ??
    suggestedFiles[0]?.path
  if (nextFocusPath) state.selectFile(nextFocusPath)
}

export function handleCreateFile(state: SkillEditorPageControllerActionsState) {
  state.setPendingCreate({ kind: 'file', parentPath: state.getSelectedTreePathParent() })
}

export function handleCreateFolder(state: SkillEditorPageControllerActionsState) {
  state.setPendingCreate({ kind: 'folder', parentPath: state.getSelectedTreePathParent() })
}

export function handleCreateCommit(
  state: SkillEditorPageControllerActionsState,
  fullPath: string,
  kind: 'file' | 'folder',
) {
  state.setPendingCreate(null)
  try {
    if (kind === 'file') {
      state.setDraftFiles(
        addDraftTextFile(state.getDraftFiles(), state.getEmptyDirectoryPaths(), fullPath),
      )
      const nextFile = state.getDraftFiles().at(-1)
      if (nextFile) state.selectFile(nextFile.path)
      return
    }
    state.setEmptyDirectoryPaths(
      addEmptyDirectory(state.getEmptyDirectoryPaths(), state.getDraftFiles(), fullPath),
    )
    state.setSelectedTreePath(normalizeSkillBundlePath(fullPath))
    state.setSelectedTreeKind('directory')
  } catch (err) {
    toastStore.error(err instanceof Error ? err.message : `Failed to create ${kind}.`)
  }
}

export function handleCreateCancel(state: SkillEditorPageControllerActionsState) {
  if (state.getPendingCreate()) state.setPendingCreate(null)
}

export function handleRenameNode(
  state: SkillEditorPageControllerActionsState,
  oldPath: string,
  newPath: string,
  kind: SkillTreeKind,
) {
  try {
    const normalizedNextPath = normalizeSkillBundlePath(newPath)
    if (kind === 'file') {
      state.setDraftFiles(renameFilePath(state.getDraftFiles(), oldPath, normalizedNextPath))
    } else {
      const renamed = renameDirectoryPath(
        state.getDraftFiles(),
        state.getEmptyDirectoryPaths(),
        oldPath,
        normalizedNextPath,
      )
      state.setDraftFiles(renamed.files)
      state.setEmptyDirectoryPaths(renamed.emptyDirectoryPaths)
    }
    replaceOpenPathPrefix(state, oldPath, normalizedNextPath)
    state.setSelectedTreePath(normalizedNextPath)
    state.setSelectedTreeKind(kind)
    if (kind === 'file') state.setSelectedFilePath(normalizedNextPath)
  } catch (err) {
    toastStore.error(err instanceof Error ? err.message : 'Failed to rename.')
  }
}

export function handleDeleteNode(
  state: SkillEditorPageControllerActionsState,
  path: string,
  kind: SkillTreeKind,
) {
  const label = kind === 'directory' ? 'folder' : 'file'
  if (!window.confirm(`Delete ${label} "${path}"?`)) return

  try {
    if (kind === 'file') {
      state.setDraftFiles(deleteFilePath(state.getDraftFiles(), path))
    } else {
      const deleted = deleteDirectoryPath(
        state.getDraftFiles(),
        state.getEmptyDirectoryPaths(),
        path,
      )
      state.setDraftFiles(deleted.files)
      state.setEmptyDirectoryPaths(deleted.emptyDirectoryPaths)
    }
    removeOpenPathsUnder(state, path)
  } catch (err) {
    toastStore.error(err instanceof Error ? err.message : `Failed to delete ${label}.`)
  }
}

export async function handleSave(
  state: SkillEditorPageControllerActionsState,
  skillId: string,
  projectId?: string,
) {
  const skill = state.getSkill()
  if (!skill) return

  const entrypointContent = currentEntrypointContent(state)
  if (!entrypointContent.trim()) {
    toastStore.error('Skill content is required.')
    return
  }
  if (!state.getHasDirtyChanges()) {
    toastStore.warning(
      state.getEmptyDraftDirectories().length > 0
        ? 'Empty folders are not persisted until they contain at least one file.'
        : 'No changes to save.',
    )
    return
  }
  if (state.getEmptyDraftDirectories().length > 0) {
    toastStore.warning('Empty folders are not persisted until they contain at least one file.')
  }

  try {
    await updateSkill(skill.id, {
      description: state.getEditDescription().trim(),
      content: entrypointContent,
      files: buildBundleRequestFiles(state),
    })
    applySavedSkillLoad(state, await loadSkillEditorData(skillId, projectId))
    toastStore.success(`Saved ${skill.name}.`)
  } catch (err) {
    toastStore.error(err instanceof ApiError ? err.detail : 'Failed to save skill.')
  }
}
