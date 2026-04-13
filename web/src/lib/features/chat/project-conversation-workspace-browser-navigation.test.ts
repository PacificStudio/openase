import { cleanup, fireEvent, render, waitFor, within } from '@testing-library/svelte'
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'

import type { ProjectConversationWorkspaceDiff } from '$lib/api/chat'
const {
  checkoutProjectConversationWorkspaceBranch,
  commitProjectConversationWorkspace,
  createProjectConversationWorkspaceFile,
  discardProjectConversationWorkspaceFile,
  deleteProjectConversationWorkspaceFile,
  getProjectConversationWorkspaceGitGraph,
  getProjectConversationWorkspace,
  getProjectConversationWorkspaceDiff,
  getProjectConversationWorkspaceRepoRefs,
  getProjectConversationWorkspaceFilePatch,
  getProjectConversationWorkspaceFilePreview,
  listProjectConversationWorkspaceTree,
  renameProjectConversationWorkspaceFile,
  saveProjectConversationWorkspaceFile,
  searchProjectConversationWorkspacePaths,
  stageAllProjectConversationWorkspaceFiles,
  stageProjectConversationWorkspaceFile,
  unstageProjectConversationWorkspace,
} = vi.hoisted(() => ({
  checkoutProjectConversationWorkspaceBranch: vi.fn(),
  commitProjectConversationWorkspace: vi.fn(),
  createProjectConversationWorkspaceFile: vi.fn(),
  discardProjectConversationWorkspaceFile: vi.fn(),
  deleteProjectConversationWorkspaceFile: vi.fn(),
  getProjectConversationWorkspaceGitGraph: vi.fn(),
  getProjectConversationWorkspace: vi.fn(),
  getProjectConversationWorkspaceDiff: vi.fn(),
  getProjectConversationWorkspaceRepoRefs: vi.fn(),
  getProjectConversationWorkspaceFilePatch: vi.fn(),
  getProjectConversationWorkspaceFilePreview: vi.fn(),
  listProjectConversationWorkspaceTree: vi.fn(),
  renameProjectConversationWorkspaceFile: vi.fn(),
  saveProjectConversationWorkspaceFile: vi.fn(),
  searchProjectConversationWorkspacePaths: vi.fn(),
  stageAllProjectConversationWorkspaceFiles: vi.fn(),
  stageProjectConversationWorkspaceFile: vi.fn(),
  unstageProjectConversationWorkspace: vi.fn(),
}))

vi.mock('$lib/api/chat', () => ({
  checkoutProjectConversationWorkspaceBranch,
  commitProjectConversationWorkspace,
  createProjectConversationWorkspaceFile,
  discardProjectConversationWorkspaceFile,
  deleteProjectConversationWorkspaceFile,
  getProjectConversationWorkspaceGitGraph,
  getProjectConversationWorkspace,
  getProjectConversationWorkspaceDiff,
  getProjectConversationWorkspaceRepoRefs,
  getProjectConversationWorkspaceFilePatch,
  getProjectConversationWorkspaceFilePreview,
  listProjectConversationWorkspaceTree,
  renameProjectConversationWorkspaceFile,
  saveProjectConversationWorkspaceFile,
  searchProjectConversationWorkspacePaths,
  stageAllProjectConversationWorkspaceFiles,
  stageProjectConversationWorkspaceFile,
  unstageProjectConversationWorkspace,
}))

import ProjectConversationWorkspaceBrowser from './project-conversation-workspace-browser.svelte'
import {
  ensureResizeObserver,
  mockWorkspaceMetadata,
  workspaceDiff,
} from './project-conversation-workspace-browser.test-helpers'
import { workspaceFileDraftStorageKey } from './project-conversation-workspace-file-drafts'

function buildTextPreview(path: string, content: string) {
  return {
    conversationId: 'conversation-1',
    repoPath: 'services/openase',
    path,
    sizeBytes: content.length,
    mediaType: 'text/plain',
    previewKind: 'text' as const,
    truncated: false,
    content,
    revision: `rev-${path}`,
    writable: true,
    readOnlyReason: '',
    encoding: 'utf-8' as const,
    lineEnding: 'lf' as const,
  }
}

describe('ProjectConversationWorkspaceBrowser', () => {
  beforeAll(() => {
    ensureResizeObserver()
  })

  beforeEach(() => {
    getProjectConversationWorkspaceRepoRefs.mockResolvedValue({
      repoRefs: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        currentRef: {
          kind: 'branch',
          displayName: 'agent/conv-123',
          cacheKey: 'branch:refs/heads/agent/conv-123',
          branchName: 'agent/conv-123',
          branchFullName: 'refs/heads/agent/conv-123',
          commitId: '123456789abc',
          shortCommitId: '123456789abc',
          subject: 'Add workspace browser scaffolding',
        },
        localBranches: [],
        remoteBranches: [],
      },
    })
    getProjectConversationWorkspaceGitGraph.mockResolvedValue({
      gitGraph: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        limit: 40,
        commits: [],
      },
    })
    checkoutProjectConversationWorkspaceBranch.mockResolvedValue({
      checkout: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        currentRef: {
          kind: 'branch',
          displayName: 'agent/conv-123',
          cacheKey: 'branch:refs/heads/agent/conv-123',
          branchName: 'agent/conv-123',
          branchFullName: 'refs/heads/agent/conv-123',
          commitId: '123456789abc',
          shortCommitId: '123456789abc',
          subject: 'Add workspace browser scaffolding',
        },
        createdLocalBranch: '',
      },
    })
    stageProjectConversationWorkspaceFile.mockResolvedValue({
      conversationId: 'conversation-1',
      repoPath: 'services/openase',
      path: 'README.md',
    })
    stageAllProjectConversationWorkspaceFiles.mockResolvedValue({
      conversationId: 'conversation-1',
      repoPath: 'services/openase',
    })
    commitProjectConversationWorkspace.mockResolvedValue({
      conversationId: 'conversation-1',
      repoPath: 'services/openase',
      output: '[agent/conv-123 1234567] feat: test',
    })
    unstageProjectConversationWorkspace.mockResolvedValue({
      conversationId: 'conversation-1',
      repoPath: 'services/openase',
      path: '',
    })
    discardProjectConversationWorkspaceFile.mockResolvedValue({
      conversationId: 'conversation-1',
      repoPath: 'services/openase',
      path: 'README.md',
    })
  })

  afterEach(() => {
    cleanup()
    vi.clearAllMocks()
    window.localStorage.clear()
  })

  it('reloads workspace metadata after the parent diff refresh completes', async () => {
    mockWorkspaceMetadata(getProjectConversationWorkspace)
    listProjectConversationWorkspaceTree.mockResolvedValue({
      workspaceTree: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        path: '',
        entries: [{ path: 'README.md', name: 'README.md', kind: 'file', sizeBytes: 64 }],
      },
    })
    getProjectConversationWorkspaceFilePreview.mockResolvedValue({
      filePreview: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        path: 'README.md',
        sizeBytes: 64,
        mediaType: 'text/plain',
        previewKind: 'text',
        truncated: false,
        content: 'line one\nline two\n',
      },
    })
    getProjectConversationWorkspaceFilePatch.mockResolvedValue({
      filePatch: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        path: 'README.md',
        status: 'modified',
        diffKind: 'text',
        truncated: false,
        diff: '@@ -1 +1,2 @@\n line one\n+line two\n',
      },
    })

    const view = render(ProjectConversationWorkspaceBrowser, {
      props: {
        conversationId: 'conversation-1',
        workspaceDiff,
        workspaceDiffLoading: false,
      },
    })

    await waitFor(() => expect(getProjectConversationWorkspace).toHaveBeenCalledTimes(1))

    await view.rerender({
      conversationId: 'conversation-1',
      workspaceDiff,
      workspaceDiffLoading: true,
    })
    await view.rerender({
      conversationId: 'conversation-1',
      workspaceDiff,
      workspaceDiffLoading: false,
    })

    await waitFor(() => expect(getProjectConversationWorkspace).toHaveBeenCalledTimes(2))
  })

  it('lists root entries, expands directories, and shows the selected repo branch info', async () => {
    mockWorkspaceMetadata(getProjectConversationWorkspace)
    listProjectConversationWorkspaceTree.mockImplementation(async (_conversationId, input) => ({
      workspaceTree: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        path: input.path,
        entries:
          input.path === ''
            ? [
                { path: 'src', name: 'src', kind: 'directory', sizeBytes: 0 },
                { path: 'README.md', name: 'README.md', kind: 'file', sizeBytes: 64 },
              ]
            : [{ path: 'src/main.ts', name: 'main.ts', kind: 'file', sizeBytes: 42 }],
      },
    }))

    const view = render(ProjectConversationWorkspaceBrowser, {
      props: {
        conversationId: 'conversation-1',
        workspaceDiff,
        workspaceDiffLoading: false,
      },
    })

    await waitFor(() => {
      expect(view.container.textContent).not.toContain('Loading workspace…')
    })
    const explorerList = view.getByTestId('workspace-browser-explorer-list')
    await within(explorerList).findByRole('button', { name: /README\.md/ }, { timeout: 3000 })
    expect(listProjectConversationWorkspaceTree).toHaveBeenCalledWith('conversation-1', {
      repoPath: 'services/openase',
      path: '',
    })
    expect(view.container.textContent).toContain('agent/conv-123')
    expect(view.container.textContent).toContain('123456789abc')

    await fireEvent.click(view.getByRole('button', { name: 'src' }))

    await waitFor(() =>
      expect(listProjectConversationWorkspaceTree).toHaveBeenCalledWith('conversation-1', {
        repoPath: 'services/openase',
        path: 'src',
      }),
    )
    await view.findByRole('button', { name: 'main.ts' }, { timeout: 3000 })
  })

  it('loads and renders file previews when a file is selected from the explorer', async () => {
    mockWorkspaceMetadata(getProjectConversationWorkspace)
    listProjectConversationWorkspaceTree.mockResolvedValue({
      workspaceTree: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        path: '',
        entries: [{ path: 'README.md', name: 'README.md', kind: 'file', sizeBytes: 31 }],
      },
    })
    getProjectConversationWorkspaceFilePreview.mockResolvedValue({
      filePreview: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        path: 'README.md',
        sizeBytes: 31,
        mediaType: 'text/plain',
        previewKind: 'text',
        truncated: false,
        content: 'line alpha\nline beta\n',
      },
    })
    getProjectConversationWorkspaceFilePatch.mockResolvedValue({
      filePatch: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        path: 'README.md',
        status: 'modified',
        diffKind: 'binary',
        truncated: false,
        diff: '',
      },
    })

    const view = render(ProjectConversationWorkspaceBrowser, {
      props: {
        conversationId: 'conversation-1',
        workspaceDiff,
        workspaceDiffLoading: false,
      },
    })

    await waitFor(() => {
      expect(view.container.textContent).not.toContain('Loading workspace…')
    })
    const explorerList = view.getByTestId('workspace-browser-explorer-list')
    await fireEvent.click(
      await within(explorerList).findByRole('button', { name: /README\.md/ }, { timeout: 3000 }),
    )

    await waitFor(() => {
      expect(getProjectConversationWorkspaceFilePreview).toHaveBeenCalledWith('conversation-1', {
        repoPath: 'services/openase',
        path: 'README.md',
      })
      expect(getProjectConversationWorkspaceFilePatch).toHaveBeenCalledWith('conversation-1', {
        repoPath: 'services/openase',
        path: 'README.md',
      })
    })

    expect(view.container.textContent).toContain('README.md')
    expect(view.container.textContent).toContain('text/plain')
    await waitFor(() => {
      expect(view.container.textContent).toContain('line alpha')
      expect(view.container.textContent).toContain('line beta')
    })
    expect(view.getByTestId('workspace-browser-detail-panel').className).toContain(
      'overflow-hidden',
    )
    expect(view.getByTestId('workspace-browser-detail-content').className).toContain(
      'overflow-hidden',
    )
    expect(view.getByTestId('workspace-browser-detail-scroll-frame').className).toContain(
      'overflow-hidden',
    )
    expect(view.container.querySelector('.code-editor')).not.toBeNull()
    expect(view.container.querySelector('.cm-scroller')).not.toBeNull()
    expect(view.container.querySelector('.cm-lineWrapping')).not.toBeNull()
  })

  it('renders git changes and opens a changed file from the status list', async () => {
    mockWorkspaceMetadata(getProjectConversationWorkspace)
    listProjectConversationWorkspaceTree.mockResolvedValue({
      workspaceTree: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        path: '',
        entries: [],
      },
    })
    getProjectConversationWorkspaceFilePreview.mockResolvedValue({
      filePreview: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        path: 'src/new.ts',
        sizeBytes: 55,
        mediaType: 'text/plain',
        previewKind: 'text',
        truncated: false,
        content: 'export const ready = true;\n',
      },
    })
    getProjectConversationWorkspaceFilePatch.mockResolvedValue({
      filePatch: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        path: 'src/new.ts',
        status: 'added',
        diffKind: 'text',
        truncated: false,
        diff: '@@ -0,0 +1 @@\n+export const ready = true;\n',
      },
    })

    const gitStatusDiff = {
      ...workspaceDiff,
      repos: [
        {
          ...workspaceDiff.repos[0],
          files: [
            {
              path: 'src/new.ts',
              status: 'added',
              staged: false,
              unstaged: true,
              added: 5,
              removed: 0,
            },
          ],
        },
      ],
    } satisfies ProjectConversationWorkspaceDiff

    const view = render(ProjectConversationWorkspaceBrowser, {
      props: {
        conversationId: 'conversation-1',
        workspaceDiff: gitStatusDiff,
        workspaceDiffLoading: false,
      },
    })

    await fireEvent.click(await view.findByRole('button', { name: /1 file/i }, { timeout: 3000 }))
    const changeLabel = await view.findByText('new.ts', { exact: true }, { timeout: 3000 })
    const changeButton = changeLabel.closest('button') as HTMLButtonElement
    const changeRow = changeButton.parentElement as HTMLElement
    expect(changeRow.textContent).toContain('+5 -0')
    expect(changeRow.textContent).toContain('A')

    await fireEvent.click(changeButton)

    await waitFor(() => {
      expect(getProjectConversationWorkspaceFilePreview).toHaveBeenCalledWith('conversation-1', {
        repoPath: 'services/openase',
        path: 'src/new.ts',
      })
      expect(getProjectConversationWorkspaceFilePatch).toHaveBeenCalledWith('conversation-1', {
        repoPath: 'services/openase',
        path: 'src/new.ts',
      })
    })

    expect(view.container.textContent).toContain('added')
    await waitFor(() => {
      expect(view.container.textContent).toContain('export const ready = true;')
    })
    expect(view.getByTestId('workspace-browser-detail-panel').className).toContain(
      'overflow-hidden',
    )
    expect(view.getByTestId('workspace-browser-detail-content').className).toContain(
      'overflow-hidden',
    )
    expect(view.getByTestId('workspace-browser-detail-scroll-frame').className).toContain(
      'overflow-hidden',
    )
    expect(view.container.querySelector('.code-editor')).not.toBeNull()
  })

  it('stages all, unstages, and commits from the branch changes panel, then refreshes workspace diff state', async () => {
    mockWorkspaceMetadata(getProjectConversationWorkspace)
    listProjectConversationWorkspaceTree.mockResolvedValue({
      workspaceTree: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        path: '',
        entries: [],
      },
    })

    const initialDiff = {
      ...workspaceDiff,
      filesChanged: 2,
      added: 7,
      removed: 0,
      repos: [
        {
          ...workspaceDiff.repos[0],
          filesChanged: 2,
          added: 7,
          removed: 0,
          files: [
            {
              path: 'README.md',
              status: 'modified',
              staged: false,
              unstaged: true,
              added: 2,
              removed: 0,
            },
            {
              path: 'src/new.ts',
              status: 'added',
              staged: false,
              unstaged: true,
              added: 5,
              removed: 0,
            },
          ],
        },
      ],
    } satisfies ProjectConversationWorkspaceDiff
    const stagedDiff = {
      ...workspaceDiff,
      filesChanged: 2,
      added: 7,
      removed: 0,
      repos: [
        {
          ...workspaceDiff.repos[0],
          filesChanged: 2,
          added: 7,
          removed: 0,
          files: [
            {
              path: 'README.md',
              status: 'modified',
              staged: true,
              unstaged: false,
              added: 2,
              removed: 0,
            },
            {
              path: 'src/new.ts',
              status: 'added',
              staged: true,
              unstaged: false,
              added: 5,
              removed: 0,
            },
          ],
        },
      ],
    } satisfies ProjectConversationWorkspaceDiff
    const mixedDiff = {
      ...workspaceDiff,
      filesChanged: 2,
      added: 7,
      removed: 0,
      repos: [
        {
          ...workspaceDiff.repos[0],
          filesChanged: 2,
          added: 7,
          removed: 0,
          files: [
            {
              path: 'README.md',
              status: 'modified',
              staged: false,
              unstaged: true,
              added: 2,
              removed: 0,
            },
            {
              path: 'src/new.ts',
              status: 'added',
              staged: true,
              unstaged: false,
              added: 5,
              removed: 0,
            },
          ],
        },
      ],
    } satisfies ProjectConversationWorkspaceDiff
    const cleanDiff = {
      ...workspaceDiff,
      dirty: false,
      reposChanged: 0,
      filesChanged: 0,
      added: 0,
      removed: 0,
      repos: [],
    } satisfies ProjectConversationWorkspaceDiff

    getProjectConversationWorkspaceDiff
      .mockResolvedValueOnce({ workspaceDiff: stagedDiff })
      .mockResolvedValueOnce({ workspaceDiff: mixedDiff })
      .mockResolvedValueOnce({ workspaceDiff: stagedDiff })
      .mockResolvedValueOnce({ workspaceDiff: cleanDiff })

    const view = render(ProjectConversationWorkspaceBrowser, {
      props: {
        conversationId: 'conversation-1',
        workspaceDiff: initialDiff,
        workspaceDiffLoading: false,
      },
    })

    await fireEvent.click(await view.findByRole('button', { name: /2 files/i }, { timeout: 3000 }))
    await fireEvent.click(view.getByTestId('workspace-branch-stage-all'))

    await waitFor(() => {
      expect(stageAllProjectConversationWorkspaceFiles).toHaveBeenCalledWith('conversation-1', {
        repoPath: 'services/openase',
      })
      expect(getProjectConversationWorkspaceDiff).toHaveBeenCalledTimes(1)
      expect(view.container.textContent).toContain('Staged 2')
    })

    await fireEvent.click(view.getByTestId('workspace-branch-unstage-README.md'))

    await waitFor(() => {
      expect(unstageProjectConversationWorkspace).toHaveBeenCalledWith('conversation-1', {
        repoPath: 'services/openase',
        path: 'README.md',
      })
      expect(getProjectConversationWorkspaceDiff).toHaveBeenCalledTimes(2)
    })

    await fireEvent.click(view.getByTestId('workspace-branch-stage-all'))

    await waitFor(() => {
      expect(stageAllProjectConversationWorkspaceFiles).toHaveBeenCalledTimes(2)
      expect(getProjectConversationWorkspaceDiff).toHaveBeenCalledTimes(3)
      expect(view.container.textContent).toContain('Staged 2')
    })

    await fireEvent.input(view.getByTestId('workspace-branch-commit-message'), {
      target: { value: 'feat: stage browser test' },
    })
    await fireEvent.click(view.getByTestId('workspace-branch-commit-button'))

    await waitFor(() => {
      expect(commitProjectConversationWorkspace).toHaveBeenCalledWith('conversation-1', {
        repoPath: 'services/openase',
        message: 'feat: stage browser test',
      })
      expect(getProjectConversationWorkspaceDiff).toHaveBeenCalledTimes(4)
      expect(view.container.textContent).toContain('clean')
    })
  })

  it('expands the explorer tree to the selected changed file from the changes list', async () => {
    mockWorkspaceMetadata(getProjectConversationWorkspace)
    listProjectConversationWorkspaceTree.mockImplementation(async (_conversationId, input) => ({
      workspaceTree: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        path: input.path,
        entries:
          input.path === ''
            ? [
                { path: 'src', name: 'src', kind: 'directory', sizeBytes: 0 },
                { path: 'README.md', name: 'README.md', kind: 'file', sizeBytes: 64 },
              ]
            : [{ path: 'src/new.ts', name: 'new.ts', kind: 'file', sizeBytes: 55 }],
      },
    }))
    getProjectConversationWorkspaceFilePreview.mockResolvedValue({
      filePreview: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        path: 'src/new.ts',
        sizeBytes: 55,
        mediaType: 'text/plain',
        previewKind: 'text',
        truncated: false,
        content: 'export const ready = true;\n',
      },
    })
    getProjectConversationWorkspaceFilePatch.mockResolvedValue({
      filePatch: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        path: 'src/new.ts',
        status: 'added',
        diffKind: 'text',
        truncated: false,
        diff: '@@ -0,0 +1 @@\n+export const ready = true;\n',
      },
    })

    const gitStatusDiff = {
      ...workspaceDiff,
      repos: [
        {
          ...workspaceDiff.repos[0],
          files: [
            {
              path: 'src/new.ts',
              status: 'added',
              staged: false,
              unstaged: true,
              added: 5,
              removed: 0,
            },
          ],
        },
      ],
    } satisfies ProjectConversationWorkspaceDiff

    const view = render(ProjectConversationWorkspaceBrowser, {
      props: {
        conversationId: 'conversation-1',
        workspaceDiff: gitStatusDiff,
        workspaceDiffLoading: false,
      },
    })

    await fireEvent.click(await view.findByRole('button', { name: /1 file/i }, { timeout: 3000 }))
    const changeLabel = await view.findByText('new.ts', { exact: true }, { timeout: 3000 })
    const changeButton = changeLabel.closest('button') as HTMLButtonElement
    const explorerList = view.getByTestId('workspace-browser-explorer-list')
    expect(within(explorerList).queryAllByRole('button', { name: /new\.ts/i })).toHaveLength(0)

    await fireEvent.click(changeButton)

    await waitFor(() => {
      expect(listProjectConversationWorkspaceTree).toHaveBeenCalledWith('conversation-1', {
        repoPath: 'services/openase',
        path: 'src',
      })
      expect(within(explorerList).getAllByRole('button', { name: /new\.ts/i })).toHaveLength(1)
    })

    expect(view.container.textContent).toContain('src')
    await waitFor(() => {
      expect(view.container.textContent).toContain('export const ready = true;')
    })
  })

  it('asks before closing a dirty tab and discards the persisted draft when requested', async () => {
    mockWorkspaceMetadata(getProjectConversationWorkspace)
    getProjectConversationWorkspaceDiff.mockResolvedValue({ workspaceDiff })
    listProjectConversationWorkspaceTree.mockResolvedValue({
      workspaceTree: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        path: '',
        entries: [{ path: 'README.md', name: 'README.md', kind: 'file', sizeBytes: 64 }],
      },
    })
    getProjectConversationWorkspaceFilePreview.mockResolvedValue({
      filePreview: buildTextPreview('README.md', 'line one\n'),
    })
    getProjectConversationWorkspaceFilePatch.mockResolvedValue({
      filePatch: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        path: 'README.md',
        status: 'modified',
        diffKind: 'text',
        truncated: false,
        diff: '@@ -1 +1 @@\n-line one\n+line one\n',
      },
    })
    saveProjectConversationWorkspaceFile.mockResolvedValue({
      file: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        path: 'README.md',
        revision: 'rev-readme-next',
        sizeBytes: 18,
        encoding: 'utf-8',
        lineEnding: 'lf',
      },
    })

    const persistedKey = workspaceFileDraftStorageKey({
      conversationId: 'conversation-1',
      repoPath: 'services/openase',
      filePath: 'README.md',
      refCacheKey: 'branch:refs/heads/agent/conv-123',
    })
    window.localStorage.setItem(
      'openase.project-conversation.workspace-file-drafts',
      JSON.stringify({
        [persistedKey]: {
          draftContent: 'line one\nline two\n',
          baseSavedContent: 'line one\n',
          baseSavedRevision: 'rev-README.md',
          encoding: 'utf-8',
          lineEnding: 'lf',
          updatedAt: '2026-04-11T00:00:00.000Z',
        },
      }),
    )

    const view = render(ProjectConversationWorkspaceBrowser, {
      props: {
        conversationId: 'conversation-1',
        workspaceDiff,
        workspaceDiffLoading: false,
      },
    })

    await fireEvent.click(
      await view.findByRole('button', { name: /README\.md/ }, { timeout: 3000 }),
    )

    await waitFor(() => {
      expect(view.getByTestId('workspace-browser-detail-tab-dirty-dot')).toBeTruthy()
      expect(view.container.textContent).toContain('line two')
    })

    await fireEvent.click(view.getByLabelText('Close README.md'))

    expect(await view.findByText('Save changes?')).toBeTruthy()
    expect(
      await view.findByText('README.md has unsaved changes. Save them before closing the tab?'),
    ).toBeTruthy()

    await fireEvent.click(view.getByRole('button', { name: "Don't save" }))

    await waitFor(() => {
      expect(view.queryByTestId('workspace-browser-detail-tab-README.md')).toBeNull()
      expect(
        window.localStorage.getItem('openase.project-conversation.workspace-file-drafts'),
      ).toBeNull()
    })
  })

  it('prompts the browser before unload when any open tab has unsaved changes', async () => {
    mockWorkspaceMetadata(getProjectConversationWorkspace)
    getProjectConversationWorkspaceDiff.mockResolvedValue({ workspaceDiff })
    listProjectConversationWorkspaceTree.mockResolvedValue({
      workspaceTree: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        path: '',
        entries: [{ path: 'README.md', name: 'README.md', kind: 'file', sizeBytes: 64 }],
      },
    })
    getProjectConversationWorkspaceFilePreview.mockResolvedValue({
      filePreview: buildTextPreview('README.md', 'line one\n'),
    })
    getProjectConversationWorkspaceFilePatch.mockResolvedValue({
      filePatch: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        path: 'README.md',
        status: 'modified',
        diffKind: 'text',
        truncated: false,
        diff: '@@ -1 +1 @@\n-line one\n+line one\n',
      },
    })

    const persistedKey = workspaceFileDraftStorageKey({
      conversationId: 'conversation-1',
      repoPath: 'services/openase',
      filePath: 'README.md',
      refCacheKey: 'branch:refs/heads/agent/conv-123',
    })
    window.localStorage.setItem(
      'openase.project-conversation.workspace-file-drafts',
      JSON.stringify({
        [persistedKey]: {
          draftContent: 'dirty readme\n',
          baseSavedContent: 'line one\n',
          baseSavedRevision: 'rev-README.md',
          encoding: 'utf-8',
          lineEnding: 'lf',
          updatedAt: '2026-04-11T00:00:00.000Z',
        },
      }),
    )

    const view = render(ProjectConversationWorkspaceBrowser, {
      props: {
        conversationId: 'conversation-1',
        workspaceDiff,
        workspaceDiffLoading: false,
      },
    })

    await fireEvent.click(
      await view.findByRole('button', { name: /README\.md/ }, { timeout: 3000 }),
    )
    await waitFor(() =>
      expect(view.getByTestId('workspace-browser-detail-tab-dirty-dot')).toBeTruthy(),
    )

    const event = new Event('beforeunload', { cancelable: true })
    window.dispatchEvent(event)
    expect(event.defaultPrevented).toBe(true)

    view.unmount()
  })
})
