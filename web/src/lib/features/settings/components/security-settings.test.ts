import { cleanup, render } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { appStore } from '$lib/stores/app.svelte'
import SecuritySettings from './security-settings.svelte'

const { getSecuritySettings } = vi.hoisted(() => ({
  getSecuritySettings: vi.fn(),
}))

vi.mock('$lib/api/openase', () => ({
  getSecuritySettings,
}))

describe('Security settings', () => {
  afterEach(() => {
    cleanup()
    appStore.currentProject = null
    vi.clearAllMocks()
  })

  it('renders the shipped security posture and explicit deferred scope', async () => {
    appStore.currentProject = {
      id: '9f34ff64-f08b-4a06-b555-f47b34957860',
      organization_id: 'org-1',
      name: 'Atlas',
      slug: 'atlas',
      description: '',
      status: 'active',
      default_workflow_id: null,
      default_agent_provider_id: null,
      accessible_machine_ids: [],
      max_concurrent_agents: 4,
    }

    getSecuritySettings.mockResolvedValue({
      security: {
        project_id: appStore.currentProject.id,
        agent_tokens: {
          transport: 'Bearer token',
          environment_variable: 'OPENASE_AGENT_TOKEN',
          token_prefix: 'ase_agent_',
          default_scopes: ['tickets.create', 'tickets.list'],
          supported_project_scopes: ['projects.update', 'projects.add_repo'],
        },
        webhooks: {
          legacy_github_endpoint: 'POST /api/v1/webhooks/github',
          connector_endpoint: 'POST /api/v1/webhooks/:connector/:provider',
          legacy_github_signature_required: true,
        },
        secret_hygiene: {
          notification_channel_configs_redacted: true,
        },
        deferred: [
          {
            key: 'human-auth',
            title: 'Human sign-in and OIDC',
            summary: 'Deferred for a later control-plane surface.',
          },
        ],
      },
    })

    const { findByText } = render(SecuritySettings)

    expect(await findByText('Agent runtime access')).toBeTruthy()
    expect(await findByText('OPENASE_AGENT_TOKEN')).toBeTruthy()
    expect(await findByText('POST /api/v1/webhooks/github')).toBeTruthy()
    expect(await findByText('Response-safe notification configs')).toBeTruthy()
    expect(await findByText('Human sign-in and OIDC')).toBeTruthy()
  })
})
