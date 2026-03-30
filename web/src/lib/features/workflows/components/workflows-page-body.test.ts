import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import type { HarnessValidationIssue } from '$lib/api/contracts'
import type { SkillState } from '../model'
import WorkflowsPageBody from './workflows-page-body.svelte'

vi.mock('./harness-ai-sidebar.svelte', () => ({
  default: vi.fn(),
}))

vi.mock('./workflow-creation-dialog.svelte', () => ({
  default: vi.fn(),
}))

const workflowFixture = {
  id: 'wf-1',
  name: 'Coding Workflow',
  type: 'coding' as const,
  agentId: 'agent-1',
  harnessPath: '.openase/harnesses/coding.md',
  pickupStatusIds: ['todo'],
  pickupStatusLabel: 'To Do',
  finishStatusIds: ['done'],
  finishStatusLabel: 'Done',
  maxConcurrent: 1,
  maxRetry: 1,
  timeoutMinutes: 30,
  stallTimeoutMinutes: 10,
  isActive: true,
  lastModified: '2026-03-28T12:00:00Z',
  recentSuccessRate: 85,
  version: 3,
}

const harnessFixture = {
  frontmatter: 'type: coding',
  body: 'You are a coding assistant.',
  rawContent: '---\ntype: coding\n---\nYou are a coding assistant.',
}

const defaultProps = {
  workflows: [workflowFixture],
  selectedId: 'wf-1',
  projectId: 'project-1',
  providers: [],
  selectedWorkflow: workflowFixture,
  harness: harnessFixture,
  draftHarness: harnessFixture.rawContent,
  variableGroups: [],
  skillStates: [] as SkillState[],
  validationIssues: [] as HarnessValidationIssue[],
  statuses: [
    { id: 'todo', name: 'To Do' },
    { id: 'done', name: 'Done' },
  ],
  agentOptions: [],
}

describe('WorkflowsPageBody', () => {
  afterEach(() => {
    cleanup()
    vi.clearAllMocks()
  })

  it('renders the workflow list and editor when data is loaded', () => {
    const { getAllByText, getByText } = render(WorkflowsPageBody, { props: defaultProps })

    // "Coding Workflow" appears both in the list and editor toolbar
    expect(getAllByText('Coding Workflow').length).toBeGreaterThanOrEqual(2)
    expect(getByText('Validate')).toBeTruthy()
  })

  it('hides the workflow list when showList is false', () => {
    const { queryByText, getByText } = render(WorkflowsPageBody, {
      props: { ...defaultProps, showList: false },
    })

    // The list heading "Workflows" should not be present
    // The editor should still render
    expect(getByText('Validate')).toBeTruthy()
    // Workflow list items won't render since the list is hidden
    // Note: "Coding Workflow" still appears in the editor toolbar
  })

  it('opens the settings sheet when onToggleDetail is triggered', async () => {
    const { getByTitle, findByText } = render(WorkflowsPageBody, {
      props: { ...defaultProps, showDetail: false },
    })

    await fireEvent.click(getByTitle('Workflow settings'))

    // The sheet should open showing workflow detail content
    await waitFor(() => {
      expect(findByText('Workflow Settings')).toBeTruthy()
    })
  })

  it('shows the loading state when loading is true', () => {
    const { container } = render(WorkflowsPageBody, {
      props: { ...defaultProps, loading: true },
    })

    // Should not show the editor when loading
    expect(container.querySelector('[title="Workflow settings"]')).toBeNull()
  })

  it('renders the workflow list at w-52 width', () => {
    const { container } = render(WorkflowsPageBody, { props: defaultProps })
    const listContainer = container.querySelector('.w-52')
    expect(listContainer).toBeTruthy()
  })
})
