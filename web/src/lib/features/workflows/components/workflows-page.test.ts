import { cleanup, fireEvent, render, waitFor, within } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { ApiError } from '$lib/api/client'
import type { Organization, Project } from '$lib/api/contracts'
import { appStore } from '$lib/stores/app.svelte'
import WorkflowsPage from './workflows-page.svelte'

const {
  loadWorkflowPageData,
  loadWorkflowHarness,
  saveWorkflowHarness,
  validateHarness,
  bindWorkflowSkills,
  unbindWorkflowSkills,
  listBuiltinRoles,
  getBuiltinRole,
  getScopeGroups,
} = vi.hoisted(() => ({
  loadWorkflowPageData: vi.fn(),
  loadWorkflowHarness: vi.fn(),
  saveWorkflowHarness: vi.fn(),
  validateHarness: vi.fn(),
  bindWorkflowSkills: vi.fn(),
  unbindWorkflowSkills: vi.fn(),
  listBuiltinRoles: vi.fn(),
  getBuiltinRole: vi.fn(),
  getScopeGroups: vi.fn(),
}))

vi.mock('../data', () => ({
  loadWorkflowPageData,
  loadWorkflowHarness,
}))

vi.mock('$lib/api/openase', () => ({
  saveWorkflowHarness,
  validateHarness,
  bindWorkflowSkills,
  unbindWorkflowSkills,
  listBuiltinRoles,
  getBuiltinRole,
  getScopeGroups,
}))

const { toastStore } = vi.hoisted(() => ({
  toastStore: {
    success: vi.fn(),
    error: vi.fn(),
    warning: vi.fn(),
    info: vi.fn(),
  },
}))

vi.mock('$lib/stores/toast.svelte', () => ({
  toastStore,
}))

vi.mock('./workflow-creation-dialog.svelte', () => ({
  default: vi.fn(),
}))

const orgFixture: Organization = {
  id: 'org-1',
  name: 'Acme',
  slug: 'acme',
  status: 'active',
  default_agent_provider_id: null,
}

const projectFixture: Project = {
  id: 'project-1',
  organization_id: 'org-1',
  name: 'TestProject',
  slug: 'test-project',
  description: '',
  status: 'active',
  default_agent_provider_id: null,
  accessible_machine_ids: [],
  max_concurrent_agents: 4,
}

const pageDataFixture = {
  workflows: [
    {
      id: 'wf-1',
      name: 'Coding Workflow',
      type: 'coding',
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
      history: [
        {
          id: 'wf-1-v3',
          version: 3,
          createdBy: 'user:manual',
          createdAt: '2026-03-28T12:00:00Z',
        },
      ],
    },
  ],
  selectedWorkflowId: 'wf-1',
  agentOptions: [],
  skillStates: [
    {
      name: 'lint',
      description: 'Run linters',
      path: '.openase/skills/lint/SKILL.md',
      bound: false,
    },
  ],
  builtinRoleContent: '',
  statuses: [
    { id: 'todo', name: 'To Do' },
    { id: 'done', name: 'Done' },
  ],
  variableGroups: [],
  harness: {
    frontmatter: 'type: coding',
    body: 'You are a coding assistant.',
    rawContent: '---\ntype: coding\n---\nYou are a coding assistant.',
  },
}

describe('WorkflowsPage', () => {
  beforeEach(() => {
    getScopeGroups.mockResolvedValue([])
  })

  afterEach(() => {
    cleanup()
    appStore.currentOrg = null
    appStore.currentProject = null
    vi.clearAllMocks()
  })

  async function openSkillsDropdown() {
    await waitFor(() => {
      const buttons = Array.from(document.body.querySelectorAll('button')).filter((button) =>
        button.textContent?.includes('Skills'),
      )
      expect(buttons.length).toBeGreaterThan(0)
    })
    const trigger = Array.from(document.body.querySelectorAll('button')).find((button) =>
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

  it('renders the workflows page and loads data', async () => {
    appStore.currentOrg = orgFixture
    appStore.currentProject = projectFixture

    loadWorkflowPageData.mockResolvedValue(pageDataFixture)

    const { findAllByText, findByText } = render(WorkflowsPage)

    // "Coding Workflow" appears in both the list and editor toolbar
    expect((await findAllByText('Coding Workflow')).length).toBeGreaterThanOrEqual(1)
    expect(await findByText('Validate')).toBeTruthy()
  })

  it('opens builtin workflow templates from the page header', async () => {
    appStore.currentOrg = orgFixture
    appStore.currentProject = projectFixture

    loadWorkflowPageData.mockResolvedValue(pageDataFixture)
    listBuiltinRoles.mockResolvedValue({ roles: [] })

    const { findAllByText, findByRole, findByText, findByTestId } = render(WorkflowsPage)

    expect((await findAllByText('Coding Workflow')).length).toBeGreaterThanOrEqual(1)

    await fireEvent.click(await findByRole('button', { name: 'Templates' }))

    expect(await findByTestId('workflow-template-gallery')).toBeTruthy()
    expect(await findByText('Workflow Templates')).toBeTruthy()
  })

  it('updates harness content after successful skill bind', async () => {
    appStore.currentOrg = orgFixture
    appStore.currentProject = projectFixture

    loadWorkflowPageData.mockResolvedValue(pageDataFixture)
    loadWorkflowHarness.mockResolvedValue({
      harness: {
        frontmatter: 'type: coding',
        body: 'You are a coding assistant.',
        rawContent: '---\ntype: coding\nskills:\n  - lint\n---\nYou are a coding assistant.',
      },
      history: [
        {
          id: 'wf-1-v4',
          version: 4,
          createdBy: 'user:manual',
          createdAt: '2026-03-29T12:00:00Z',
        },
      ],
      skillStates: [
        {
          name: 'lint',
          description: 'Run linters',
          path: '.openase/skills/lint/SKILL.md',
          bound: true,
        },
      ],
    })

    bindWorkflowSkills.mockResolvedValue({
      harness: {
        content: '---\ntype: coding\nskills:\n  - lint\n---\nYou are a coding assistant.',
        path: '.openase/harnesses/coding.md',
        version: 4,
      },
    })

    render(WorkflowsPage)

    await openSkillsDropdown()
    const lintButton = getSkillToggle('lint', 'Bind skill')
    await fireEvent.click(lintButton)

    await waitFor(() => {
      expect(bindWorkflowSkills).toHaveBeenCalledWith('wf-1', ['lint'])
      expect(loadWorkflowHarness).toHaveBeenCalledWith('project-1', 'wf-1')
      expect(toastStore.success).toHaveBeenCalledWith('Bound lint.')
    })
  })

  it('blocks skill toggle when harness has unsaved changes', async () => {
    appStore.currentOrg = orgFixture
    appStore.currentProject = projectFixture

    loadWorkflowPageData.mockResolvedValue({
      ...pageDataFixture,
      skillStates: [
        {
          name: 'lint',
          description: 'Run linters',
          path: '.openase/skills/lint/SKILL.md',
          bound: true,
        },
      ],
    })

    const { findAllByText, container } = render(WorkflowsPage)

    // Wait for page to load
    await findAllByText('Coding Workflow')

    // Simulate a draft change to make isDirty true
    // We do this by finding the textarea and typing in it
    const textarea = container.querySelector('textarea')
    if (textarea) {
      await fireEvent.input(textarea, { target: { value: 'modified content' } })
    }

    await openSkillsDropdown()
    const lintButton = getSkillToggle('lint', 'Unbind skill')
    await fireEvent.click(lintButton)

    await waitFor(() => {
      expect(toastStore.warning).toHaveBeenCalledWith(
        'Please save your harness changes before binding or unbinding skills.',
      )
      expect(unbindWorkflowSkills).not.toHaveBeenCalled()
    })
  })

  it('shows error toast when skill toggle fails', async () => {
    appStore.currentOrg = orgFixture
    appStore.currentProject = projectFixture

    loadWorkflowPageData.mockResolvedValue(pageDataFixture)
    bindWorkflowSkills.mockRejectedValue(new Error('Network error'))

    render(WorkflowsPage)

    await openSkillsDropdown()
    const lintButton = getSkillToggle('lint', 'Bind skill')
    await fireEvent.click(lintButton)

    await waitFor(() => {
      expect(toastStore.error).toHaveBeenCalledWith('Failed to update workflow skills.')
    })
  })

  it('binds by skill name even when the skill state carries a repository path', async () => {
    appStore.currentOrg = orgFixture
    appStore.currentProject = projectFixture

    loadWorkflowPageData.mockResolvedValue({
      ...pageDataFixture,
      skillStates: [
        {
          name: 'commit',
          description: 'Commit the current work.',
          path: '.openase/skills/commit/SKILL.md',
          bound: false,
        },
      ],
    })
    bindWorkflowSkills.mockResolvedValue({
      harness: {
        content: pageDataFixture.harness.rawContent,
        path: '.openase/harnesses/coding.md',
        version: 4,
      },
    })

    render(WorkflowsPage)

    await openSkillsDropdown()
    const commitButton = getSkillToggle('commit', 'Bind skill')
    await fireEvent.click(commitButton)

    await waitFor(() => {
      expect(bindWorkflowSkills).toHaveBeenCalledWith('wf-1', ['commit'])
      expect(bindWorkflowSkills).not.toHaveBeenCalledWith('wf-1', [
        '.openase/skills/commit/SKILL.md',
      ])
    })
  })

  it('renders a load error instead of crashing when workflow loading fails', async () => {
    appStore.currentOrg = orgFixture
    appStore.currentProject = projectFixture

    loadWorkflowPageData.mockRejectedValue(
      new ApiError(
        500,
        'prepare workflow repository checkout failed: platform-managed GitHub outbound credential is not configured',
      ),
    )

    const { findByText, queryByText } = render(WorkflowsPage)

    expect(
      await findByText(
        'prepare workflow repository checkout failed: platform-managed GitHub outbound credential is not configured',
      ),
    ).toBeTruthy()
    expect(queryByText('Coding Workflow')).toBeNull()
  })
})
