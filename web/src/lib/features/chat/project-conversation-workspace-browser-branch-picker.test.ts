import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import ProjectConversationWorkspaceBranchPicker from './project-conversation-workspace-browser-branch-picker.svelte'

describe('ProjectConversationWorkspaceBranchPicker', () => {
  beforeEach(() => {
    Element.prototype.scrollIntoView ??= vi.fn()
  })

  afterEach(() => {
    cleanup()
    vi.restoreAllMocks()
  })

  it('allows typing into the branch search input', async () => {
    const onCreateBranchName = vi.fn(async () => {})

    const view = render(ProjectConversationWorkspaceBranchPicker, {
      props: {
        currentRef: {
          kind: 'branch',
          displayName: 'main',
          cacheKey: 'branch:refs/heads/main',
          branchName: 'main',
          branchFullName: 'refs/heads/main',
          commitId: '123456789abc',
          shortCommitId: '123456789abc',
          subject: 'Main branch',
        },
        selectedRepo: {
          branch: 'main',
          headCommit: '123456789abc',
        },
        localBranches: [
          {
            name: 'main',
            fullName: 'refs/heads/main',
            scope: 'local_branch',
            current: true,
            commitId: '123456789abc',
            shortCommitId: '123456789abc',
            subject: 'Main branch',
            upstreamName: 'origin/main',
            ahead: 0,
            behind: 0,
            suggestedLocalBranchName: '',
          },
          {
            name: 'feature/workspace-git',
            fullName: 'refs/heads/feature/workspace-git',
            scope: 'local_branch',
            current: false,
            commitId: 'abcdef012345',
            shortCommitId: 'abcdef012345',
            subject: 'Workspace git branch',
            upstreamName: '',
            ahead: 0,
            behind: 0,
            suggestedLocalBranchName: '',
          },
        ],
        onCreateBranchName,
      },
    })

    await fireEvent.click(view.getByText('main'))

    const input = await view.findByPlaceholderText('Branch…')
    await fireEvent.input(input, { target: { value: 'feature' } })

    await waitFor(() => expect((input as HTMLInputElement).value).toBe('feature'))
    expect(view.getByText('feature/workspace-git')).toBeTruthy()
    expect(view.getByTestId('workspace-branch-create')).toBeTruthy()
    expect(view.getByText('Create branch "feature"')).toBeTruthy()

    await fireEvent.input(input, { target: { value: 'test' } })
    await fireEvent.keyDown(input, { key: 'Enter', code: 'Enter' })

    await waitFor(() => expect(onCreateBranchName).toHaveBeenCalledWith('test'))
  })

  it('shows staged state and routes stage, discard, and commit actions', async () => {
    const onStageFile = vi.fn(async () => {})
    const onStageAll = vi.fn(async () => {})
    const onUnstage = vi.fn(async () => {})
    const onDiscardFile = vi.fn(async () => {})
    const onCommitRepo = vi.fn(async () => {})

    const view = render(ProjectConversationWorkspaceBranchPicker, {
      props: {
        currentRef: {
          kind: 'branch',
          displayName: 'main',
          cacheKey: 'branch:refs/heads/main',
          branchName: 'main',
          branchFullName: 'refs/heads/main',
          commitId: '123456789abc',
          shortCommitId: '123456789abc',
          subject: 'Main branch',
        },
        selectedRepo: {
          branch: 'main',
          headCommit: '123456789abc',
        },
        selectedRepoDiff: {
          name: 'openase',
          path: 'services/openase',
          branch: 'main',
          dirty: true,
          filesChanged: 2,
          added: 4,
          removed: 1,
          files: [
            {
              path: 'README.md',
              status: 'modified',
              staged: false,
              unstaged: true,
              added: 1,
              removed: 1,
            },
            {
              path: 'src/app.ts',
              status: 'modified',
              staged: true,
              unstaged: false,
              added: 3,
              removed: 0,
            },
          ],
        },
        onStageFile,
        onStageAll,
        onUnstage,
        onDiscardFile,
        onCommitRepo,
      },
    })

    await fireEvent.click(view.getByText(/2 files/))

    expect(view.getByText('Staged')).toBeTruthy()
    expect(view.getByTestId('workspace-branch-stage-all')).toBeTruthy()
    expect(view.getByTestId('workspace-branch-unstage-all')).toBeTruthy()
    expect(await view.findByTestId('workspace-branch-stage-README.md')).toBeTruthy()
    expect(view.getByTestId('workspace-branch-unstage-src/app.ts')).toBeTruthy()
    expect(view.getByTestId('workspace-branch-discard-README.md')).toBeTruthy()

    await fireEvent.click(view.getByTestId('workspace-branch-stage-all'))
    expect(onStageAll).toHaveBeenCalledTimes(1)

    await fireEvent.click(view.getByTestId('workspace-branch-stage-README.md'))
    expect(onStageFile).toHaveBeenCalledWith('README.md')

    await fireEvent.click(view.getByTestId('workspace-branch-unstage-src/app.ts'))
    expect(onUnstage).toHaveBeenCalledWith('src/app.ts')

    await fireEvent.click(view.getByTestId('workspace-branch-unstage-all'))
    expect(onUnstage).toHaveBeenCalledWith(undefined)

    await fireEvent.click(view.getByTestId('workspace-branch-discard-README.md'))
    expect(onDiscardFile).toHaveBeenCalledWith('README.md')

    const input = view.getByTestId('workspace-branch-commit-message')
    await fireEvent.input(input, { target: { value: 'feat: save staged work' } })
    await fireEvent.click(view.getByTestId('workspace-branch-commit-button'))

    expect(onCommitRepo).toHaveBeenCalledWith('feat: save staged work')
  })
})
