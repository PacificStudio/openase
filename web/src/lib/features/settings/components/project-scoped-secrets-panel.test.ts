import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import ProjectScopedSecretsPanel from './project-scoped-secrets-panel.svelte'

const {
  createProjectScopedSecret,
  deleteProjectScopedSecret,
  disableProjectScopedSecret,
  listProjectScopedSecrets,
  rotateProjectScopedSecret,
} = vi.hoisted(() => ({
  createProjectScopedSecret: vi.fn(),
  deleteProjectScopedSecret: vi.fn(),
  disableProjectScopedSecret: vi.fn(),
  listProjectScopedSecrets: vi.fn(),
  rotateProjectScopedSecret: vi.fn(),
}))

vi.mock('$lib/api/openase', () => ({
  createProjectScopedSecret,
  deleteProjectScopedSecret,
  disableProjectScopedSecret,
  listProjectScopedSecrets,
  rotateProjectScopedSecret,
}))

const baseSecret = {
  kind: 'opaque',
  disabled: false,
  disabled_at: null,
  created_at: '2026-04-08T12:00:00Z',
  updated_at: '2026-04-08T12:00:00Z',
  usage_count: 1,
  usage_scopes: ['organization'],
  encryption: {
    algorithm: 'aes-256-gcm',
    key_id: 'database-dsn-sha256:v1',
    key_source: 'database_dsn_sha256',
    rotated_at: '2026-04-08T12:00:00Z',
    value_preview: 'sk-live...1234',
  },
}

describe('ProjectScopedSecretsPanel', () => {
  beforeEach(() => {
    listProjectScopedSecrets.mockResolvedValue({
      secrets: [
        {
          ...baseSecret,
          id: 'secret-org',
          organization_id: 'org-1',
          project_id: null,
          scope: 'organization',
          name: 'OPENAI_API_KEY',
          description: 'Inherited model key',
        },
      ],
    })
  })

  afterEach(() => {
    cleanup()
    vi.clearAllMocks()
  })

  it('prefills a project override from an inherited organization secret', async () => {
    const view = render(ProjectScopedSecretsPanel, {
      projectId: 'project-1',
      organizationId: 'org-1',
    })

    await fireEvent.click(await view.findByText('Use as override draft'))

    expect(await view.findByLabelText('Secret name')).toHaveProperty('value', 'OPENAI_API_KEY')
  })

  it('creates a project override through the write-only form', async () => {
    createProjectScopedSecret.mockResolvedValue({
      secret: {
        ...baseSecret,
        id: 'secret-project',
        organization_id: 'org-1',
        project_id: 'project-1',
        scope: 'project',
        name: 'OPENAI_API_KEY',
        description: 'Project-specific key',
        usage_scopes: ['project'],
      },
    })

    const view = render(ProjectScopedSecretsPanel, {
      projectId: 'project-1',
      organizationId: 'org-1',
    })

    await fireEvent.input(await view.findByLabelText('Secret name'), {
      target: { value: 'OPENAI_API_KEY' },
    })
    await fireEvent.input(await view.findByLabelText('Secret value'), {
      target: { value: 'sk-live-override' },
    })
    await fireEvent.input(await view.findByLabelText('Description'), {
      target: { value: 'Project-specific key' },
    })
    await fireEvent.click(await view.findByRole('button', { name: 'Create override' }))

    await waitFor(() => {
      expect(createProjectScopedSecret).toHaveBeenCalledWith('project-1', {
        scope: 'project',
        name: 'OPENAI_API_KEY',
        description: 'Project-specific key',
        value: 'sk-live-override',
      })
    })
  })
})
