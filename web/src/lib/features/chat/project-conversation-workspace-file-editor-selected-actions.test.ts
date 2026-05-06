import { describe, expect, it, vi } from 'vitest'

import type { WorkspaceFileEditorState } from './project-conversation-workspace-browser-state-helpers'
import { createWorkspaceFileEditorSelectedActions } from './project-conversation-workspace-file-editor-selected-actions'

function createEditorState(
  overrides: Partial<WorkspaceFileEditorState> = {},
): WorkspaceFileEditorState {
  return {
    baseSavedContent: '{"a":1}',
    baseSavedRevision: 'rev-1',
    latestSavedContent: '{"a":1}',
    latestSavedRevision: 'rev-1',
    draftContent: '{"a":1}',
    dirty: false,
    savePhase: 'idle',
    externalChange: false,
    errorMessage: '',
    encoding: 'utf-8',
    lineEnding: 'lf',
    lastSavedAt: '',
    selection: null,
    pendingPatch: null,
    ...overrides,
  }
}

describe('createWorkspaceFileEditorSelectedActions', () => {
  it('updates and formats the selected editor state', () => {
    const editorState = new Map<string, WorkspaceFileEditorState>([
      ['repo::file.json', createEditorState()],
    ])

    const actions = createWorkspaceFileEditorSelectedActions({
      getSelectedRepoPath: () => 'repo',
      getSelectedFilePath: () => 'file.json',
      getEditorState: (repoPath, filePath) =>
        editorState.get(`${repoPath ?? ''}::${filePath ?? ''}`) ?? null,
      setEditorState: (repoPath, filePath, nextState) => {
        const key = `${repoPath}::${filePath}`
        if (nextState) editorState.set(key, nextState)
        else editorState.delete(key)
      },
    })

    actions.updateSelectedDraft('{"a":1,"b":2}')
    expect(editorState.get('repo::file.json')?.dirty).toBe(true)
    expect(actions.getSelectedDraftLineDiff()).not.toBeNull()

    expect(actions.formatSelectedDocument()).toBe(true)
    expect(editorState.get('repo::file.json')?.draftContent).toBe('{\n  "a": 1,\n  "b": 2\n}')

    actions.reloadSelectedSavedVersion()
    expect(editorState.get('repo::file.json')?.draftContent).toBe('{"a":1}')
    expect(editorState.get('repo::file.json')?.dirty).toBe(false)
  })

  it('no-ops cleanly when no selected editor exists', () => {
    const setEditorState = vi.fn()
    const actions = createWorkspaceFileEditorSelectedActions({
      getSelectedRepoPath: () => '',
      getSelectedFilePath: () => '',
      getEditorState: () => null,
      setEditorState,
    })

    expect(actions.getSelectedDraftLineDiff()).toBeNull()
    expect(actions.formatSelectedDocument()).toBe(false)
    expect(actions.formatSelectedSelection()).toBe(false)

    actions.updateSelectedDraft('anything')
    actions.updateSelectedSelection({ from: 0, to: 1 })
    actions.revertSelectedDraft()
    actions.keepSelectedDraft()
    actions.discardSelectedDraft()
    actions.reloadSelectedSavedVersion()

    expect(setEditorState).not.toHaveBeenCalled()
  })
})
