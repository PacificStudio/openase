import { cleanup, render } from '@testing-library/svelte'
import { afterEach, describe, expect, it } from 'vitest'

import { capabilityCatalog } from '$lib/features/capabilities'
import { appStore } from '$lib/stores/app.svelte'
import ConnectorsSettings from './connectors-settings.svelte'

describe('Connectors settings', () => {
  afterEach(() => {
    cleanup()
    appStore.currentProject = null
  })

  it('reframes connector settings around the live runtime boundary', () => {
    appStore.currentProject = {
      id: 'project-1',
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

    const { getByText } = render(ConnectorsSettings)

    expect(capabilityCatalog.connectorsSettings.state).toBe('unwired')
    expect(getByText('Current exported surface')).toBeTruthy()
    expect(getByText('POST /api/v1/webhooks/:connector/:provider')).toBeTruthy()
    expect(getByText('Deferred management scope')).toBeTruthy()
    expect(getByText('List project connectors')).toBeTruthy()
  })
})
