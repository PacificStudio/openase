import { cleanup, fireEvent, render, within } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import type { HarnessContent, HarnessVariableGroup } from '../types'
import type { SkillState } from '../model'
import type { HarnessValidationIssue } from '$lib/api/contracts'
import WorkflowEditorPanel from './workflow-editor-panel.svelte'

vi.mock('./harness-ai-sidebar.svelte', () => ({
  default: vi.fn(),
}))

const harnessFixture: HarnessContent = {
  frontmatter: 'type: coding',
  body: 'You are a coding assistant.',
  rawContent: '---\ntype: coding\n---\nYou are a coding assistant.',
}

const workflowFixture = {
  id: 'wf-1',
  name: 'Coding Workflow',
  type: 'coding' as const,
  workflowFamily: 'coding' as const,
  classification: {
    family: 'coding' as const,
    confidence: 1,
    reasons: ['fixture'],
  },
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
  history: [],
}

const skillStatesFixture: SkillState[] = [
  {
    id: 'sk-1',
    name: 'lint',
    description: 'Run linters on changed files',
    path: 'skills/lint',
    bound: true,
  },
  {
    id: 'sk-2',
    name: 'test-runner',
    description: 'Execute test suites',
    path: 'skills/test-runner',
    bound: false,
  },
]

describe('WorkflowEditorPanel', () => {
  afterEach(() => {
    cleanup()
    vi.clearAllMocks()
  })

  function renderPanel(overrides: Record<string, unknown> = {}) {
    return render(WorkflowEditorPanel, {
      props: {
        harness: harnessFixture,
        selectedWorkflow: workflowFixture,
        skillStates: skillStatesFixture,
        validationIssues: [] as HarnessValidationIssue[],
        variableGroups: [] as HarnessVariableGroup[],
        showList: true,
        ...overrides,
      },
    })
  }

  async function openSkillsDropdown(container: HTMLElement) {
    const trigger = Array.from(container.querySelectorAll('button')).find((button) =>
      button.textContent?.includes('Skills'),
    )
    expect(trigger).toBeTruthy()
    await fireEvent.click(trigger as HTMLElement)
  }

  function getSkillToggle(skillName: string, actionTitle: 'Bind skill' | 'Unbind skill') {
    const row = within(document.body).getByText(skillName).closest('button')
    expect(row).toBeTruthy()
    return within(row as HTMLElement).getByTitle(actionTitle)
  }

  it('renders the toolbar with workflow name and action buttons', async () => {
    const { getByText, getByTitle } = renderPanel()

    expect(getByText('Coding Workflow')).toBeTruthy()
    expect(getByText('Validate')).toBeTruthy()
    expect(getByText('Save')).toBeTruthy()
    expect(getByTitle('Workflow settings')).toBeTruthy()
  })

  it('shows the unsaved badge when isDirty is true', () => {
    const { getByText } = renderPanel({ isDirty: true })
    expect(getByText('Unsaved')).toBeTruthy()
  })

  it('does not show the unsaved badge when isDirty is false', () => {
    const { queryByText } = renderPanel({ isDirty: false })
    expect(queryByText('Unsaved')).toBeNull()
  })

  it('renders skill toggles in the skills dropdown with correct bound state', async () => {
    const { container } = renderPanel()

    await openSkillsDropdown(container)

    const lintButton = getSkillToggle('lint', 'Unbind skill')
    const testButton = getSkillToggle('test-runner', 'Bind skill')

    expect(lintButton).toBeTruthy()
    expect(testButton).toBeTruthy()
  })

  it('calls onToggleSkill when a skill toggle is clicked from the dropdown', async () => {
    const onToggleSkill = vi.fn()
    const { container } = renderPanel({ onToggleSkill })

    await openSkillsDropdown(container)
    await fireEvent.click(getSkillToggle('test-runner', 'Bind skill'))

    expect(onToggleSkill).toHaveBeenCalledTimes(1)
    expect(onToggleSkill).toHaveBeenCalledWith(skillStatesFixture[1])
  })

  it('toggles the workflow list panel via the sidebar button', async () => {
    const onToggleList = vi.fn()
    const { getByTitle } = renderPanel({ onToggleList })

    await fireEvent.click(getByTitle('Hide workflow list'))
    expect(onToggleList).toHaveBeenCalledTimes(1)
  })

  it('shows the show-list icon when list is hidden', () => {
    const { getByTitle } = renderPanel({ showList: false })
    expect(getByTitle('Show workflow list')).toBeTruthy()
  })

  it('calls onToggleDetail when settings button is clicked', async () => {
    const onToggleDetail = vi.fn()
    const { getByTitle } = renderPanel({ onToggleDetail })

    await fireEvent.click(getByTitle('Workflow settings'))
    expect(onToggleDetail).toHaveBeenCalledTimes(1)
  })

  it('calls onSave and onValidate when action buttons are clicked', async () => {
    const onSave = vi.fn()
    const onValidate = vi.fn()
    const { getByText } = renderPanel({ onSave, onValidate, isDirty: true })

    await fireEvent.click(getByText('Validate'))
    expect(onValidate).toHaveBeenCalledTimes(1)

    await fireEvent.click(getByText('Save'))
    expect(onSave).toHaveBeenCalledTimes(1)
  })

  it('disables save button when not dirty', () => {
    const { getByText } = renderPanel({ isDirty: false })
    expect((getByText('Save') as HTMLButtonElement).disabled).toBe(true)
  })

  it('shows validation issues as a collapsible bar', async () => {
    const issues: HarnessValidationIssue[] = [
      { message: 'Missing type field', level: 'error', line: 1 },
      { message: 'Unknown variable', level: 'warning', line: 5 },
    ]
    const { getByText, queryByText } = renderPanel({ validationIssues: issues })

    expect(getByText('2 validation issues')).toBeTruthy()
    expect(getByText(/Missing type field/)).toBeTruthy()

    expect(queryByText('Unknown variable')).toBeNull()

    await fireEvent.click(getByText('2 validation issues'))

    expect(getByText(/Missing type field/)).toBeTruthy()
    expect(getByText(/Unknown variable/)).toBeTruthy()
  })

  it('does not show AI drawer by default', () => {
    const { container } = renderPanel()
    expect(container.querySelector('[role="separator"]')).toBeNull()
  })

  it('opens the AI drawer when AI button is clicked', async () => {
    const { getByText, container } = renderPanel()

    await fireEvent.click(getByText('AI'))

    expect(container.querySelector('[role="separator"]')).toBeTruthy()
  })

  it('shows fallback text when no workflow is selected', () => {
    const { getByText } = renderPanel({ selectedWorkflow: undefined, harness: null })
    expect(getByText('No workflow selected')).toBeTruthy()
  })

  it('displays the variable count badge when variables exist', () => {
    const variableGroups: HarnessVariableGroup[] = [
      {
        name: 'Ticket',
        variables: [
          { path: 'ticket.title', type: 'string', description: 'Ticket title' },
          { path: 'ticket.body', type: 'string', description: 'Ticket body' },
        ],
      },
    ]
    const { getByText } = renderPanel({ variableGroups })
    expect(getByText('2 vars')).toBeTruthy()
  })
})
