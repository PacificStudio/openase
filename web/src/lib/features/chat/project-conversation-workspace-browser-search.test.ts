import { fireEvent, render, waitFor } from '@testing-library/svelte'
import { describe, expect, it, vi } from 'vitest'

import type { ProjectConversationWorkspaceTreeEntry } from '$lib/api/chat'
import ProjectConversationWorkspaceBrowserSearch from './project-conversation-workspace-browser-search.svelte'

describe('ProjectConversationWorkspaceBrowserSearch', () => {
  it('shows recent files on focus and searches the repo through the backend callback', async () => {
    const onSelectFile = vi.fn()
    const onSearchPaths = vi.fn(async (query: string) => {
      if (query === 'utils') {
        return [{ path: 'deep/utils/helpers.ts', name: 'helpers.ts' }]
      }
      if (query === 'main') {
        return [{ path: 'src/main.ts', name: 'main.ts' }]
      }
      return []
    })
    const treeNodes = new Map<string, ProjectConversationWorkspaceTreeEntry[]>([
      [
        '',
        [
          { path: 'README.md', name: 'README.md', kind: 'file', sizeBytes: 64 },
          { path: 'src', name: 'src', kind: 'directory', sizeBytes: 0 },
        ],
      ],
      [
        'src',
        [
          { path: 'src/main.ts', name: 'main.ts', kind: 'file', sizeBytes: 42 },
          { path: 'src/utils.ts', name: 'utils.ts', kind: 'file', sizeBytes: 21 },
        ],
      ],
    ])

    const view = render(ProjectConversationWorkspaceBrowserSearch, {
      props: {
        selectedRepoPath: 'services/openase',
        recentFiles: [
          { repoPath: 'services/openase', filePath: 'docs/guide.md' },
          { repoPath: 'services/openase', filePath: 'src/main.ts' },
          { repoPath: 'services/other', filePath: 'other/ignore.ts' },
        ],
        treeNodes,
        onSearchPaths,
        onSelectFile,
      },
    })

    const input = view.getByTestId('workspace-browser-search-input')

    await fireEvent.focus(input)

    const recentPanel = await view.findByTestId('workspace-browser-search-panel')
    expect(recentPanel.textContent).toContain('Recent')
    expect(recentPanel.textContent).toContain('guide.md')
    expect(recentPanel.textContent).toContain('main.ts')
    expect(recentPanel.textContent).not.toContain('ignore.ts')

    await fireEvent.input(input, { target: { value: 'main' } })

    await waitFor(() => expect(view.getAllByRole('button', { name: /main\.ts/i })).toHaveLength(1))
    expect(onSearchPaths).toHaveBeenCalledWith('main', 20)
    expect(view.getByTestId('workspace-browser-search-panel').textContent).not.toContain('Recent')

    await fireEvent.input(input, { target: { value: 'utils' } })

    await waitFor(() => expect(view.getByRole('button', { name: /helpers\.ts/i })).toBeTruthy())

    await fireEvent.click(view.getByRole('button', { name: /helpers\.ts/i }))

    expect(onSelectFile).toHaveBeenCalledWith('deep/utils/helpers.ts')
    await waitFor(() => expect(view.queryByTestId('workspace-browser-search-panel')).toBeNull())
  })
})
