import { describe, expect, it } from 'vitest'

import type { ProjectConversationWorkspaceFilePreview } from '$lib/api/chat'
import type { WorkspaceFileEditorState } from './project-conversation-workspace-browser-state-helpers'
import {
  buildWorkspaceEditorWorkingSet,
  getSelectedWorkspaceDraftLineDiff,
} from './project-conversation-workspace-file-editor-derived'

function buildPreview(
  overrides: Partial<ProjectConversationWorkspaceFilePreview> = {},
): ProjectConversationWorkspaceFilePreview {
  return {
    conversationId: 'conversation-1',
    repoPath: 'services/openase',
    path: 'README.md',
    sizeBytes: 12,
    mediaType: 'text/plain',
    previewKind: 'text',
    truncated: false,
    content: 'preview content',
    revision: 'rev-1',
    writable: true,
    readOnlyReason: '',
    encoding: 'utf-8',
    lineEnding: 'lf',
    ...overrides,
  }
}

function buildEditorState(
  overrides: Partial<WorkspaceFileEditorState> = {},
): WorkspaceFileEditorState {
  return {
    baseSavedContent: 'line one\nline two\n',
    baseSavedRevision: 'rev-1',
    latestSavedContent: 'line one\nline two\n',
    latestSavedRevision: 'rev-1',
    draftContent: 'line one\nline two\n',
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

describe('buildWorkspaceEditorWorkingSet', () => {
  it('prefers draft content, falls back to preview content, and skips empty files', () => {
    const previews = new Map<string, ProjectConversationWorkspaceFilePreview>([
      ['services/openase::README.md', buildPreview({ path: 'README.md', content: 'preview readme' })],
      ['services/openase::docs/guide.md', buildPreview({ path: 'docs/guide.md', content: 'guide preview' })],
      ['services/openase::EMPTY.md', buildPreview({ path: 'EMPTY.md', content: '' })],
    ])
    const editors = new Map<string, WorkspaceFileEditorState>([
      [
        'services/openase::README.md',
        buildEditorState({ draftContent: 'draft readme', dirty: true }),
      ],
    ])

    const workingSet = buildWorkspaceEditorWorkingSet({
      recentFiles: [
        { repoPath: 'services/openase', filePath: 'README.md' },
        { repoPath: 'services/openase', filePath: 'docs/guide.md' },
        { repoPath: 'services/openase', filePath: 'EMPTY.md' },
      ],
      getEditorState: (repoPath, filePath) =>
        editors.get(`${repoPath ?? ''}::${filePath ?? ''}`) ?? null,
      getPreview: (repoPath, filePath) => previews.get(`${repoPath}::${filePath}`) ?? null,
    })

    expect(workingSet).toEqual([
      {
        filePath: 'README.md',
        contentExcerpt: 'draft readme',
        dirty: true,
        truncated: false,
      },
      {
        filePath: 'docs/guide.md',
        contentExcerpt: 'guide preview',
        dirty: false,
        truncated: false,
      },
    ])
  })
})

describe('getSelectedWorkspaceDraftLineDiff', () => {
  it('returns null when no file is selected or no editor state exists', () => {
    expect(
      getSelectedWorkspaceDraftLineDiff({
        repoPath: 'services/openase',
        filePath: '',
        getEditorState: () => null,
      }),
    ).toBeNull()

    expect(
      getSelectedWorkspaceDraftLineDiff({
        repoPath: 'services/openase',
        filePath: 'README.md',
        getEditorState: () => null,
      }),
    ).toBeNull()
  })

  it('derives line diff markers from the selected editor draft', () => {
    const editor = buildEditorState({
      latestSavedContent: 'alpha\nbeta\ngamma\n',
      draftContent: 'alpha\nbeta changed\ngamma\nextra\n',
    })

    expect(
      getSelectedWorkspaceDraftLineDiff({
        repoPath: 'services/openase',
        filePath: 'README.md',
        getEditorState: () => editor,
      }),
    ).toEqual({
      added: [4],
      modified: [2],
      deletionAbove: [],
      deletionAtEnd: false,
    })
  })
})
