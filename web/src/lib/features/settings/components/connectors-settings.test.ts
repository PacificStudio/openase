import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { capabilityCatalog } from '$lib/features/capabilities'
import { appStore } from '$lib/stores/app.svelte'
import ConnectorsSettings from './connectors-settings.svelte'

const {
  listIssueConnectors,
  createIssueConnector,
  updateIssueConnector,
  deleteIssueConnector,
  testIssueConnector,
  syncIssueConnector,
  getIssueConnectorStats,
} = vi.hoisted(() => ({
  listIssueConnectors: vi.fn(),
  createIssueConnector: vi.fn(),
  updateIssueConnector: vi.fn(),
  deleteIssueConnector: vi.fn(),
  testIssueConnector: vi.fn(),
  syncIssueConnector: vi.fn(),
  getIssueConnectorStats: vi.fn(),
}))

const { toastStore } = vi.hoisted(() => ({
  toastStore: {
    success: vi.fn(),
    error: vi.fn(),
  },
}))

vi.mock('$lib/api/openase', () => ({
  listIssueConnectors,
  createIssueConnector,
  updateIssueConnector,
  deleteIssueConnector,
  testIssueConnector,
  syncIssueConnector,
  getIssueConnectorStats,
}))

vi.mock('$lib/stores/toast.svelte', () => ({
  toastStore,
}))

describe('Connectors settings', () => {
  afterEach(() => {
    cleanup()
    appStore.currentProject = null
    vi.clearAllMocks()
  })

  it('renders live connector controls from Settings', async () => {
    appStore.currentProject = {
      id: 'project-1',
      organization_id: 'org-1',
      name: 'Atlas',
      slug: 'atlas',
      description: '',
      status: 'active',
      default_agent_provider_id: null,
      accessible_machine_ids: [],
      max_concurrent_agents: 4,
    }

    listIssueConnectors.mockResolvedValue({
      connectors: [
        {
          id: 'connector-1',
          project_id: 'project-1',
          type: 'github',
          name: 'GitHub Backend',
          status: 'active',
          config: {
            type: 'github',
            base_url: 'https://api.github.com',
            project_ref: 'acme/backend',
            poll_interval: '5m',
            sync_direction: 'bidirectional',
            filters: { labels: ['openase'], exclude_labels: [], states: [], authors: [] },
            status_mapping: { open: 'Todo', closed: 'Done' },
            auto_workflow: '',
            auth_token_configured: true,
            webhook_secret_configured: true,
          },
          last_sync_at: '2026-03-31T09:00:00Z',
          last_error: '',
          stats: { total_synced: 4, synced24h: 2, failed_count: 0 },
        },
      ],
    })
    getIssueConnectorStats.mockResolvedValue({
      stats: {
        connector_id: 'connector-1',
        status: 'active',
        last_sync_at: '2026-03-31T09:00:00Z',
        last_error: '',
        stats: { total_synced: 4, synced24h: 2, failed_count: 0 },
      },
    })
    syncIssueConnector.mockResolvedValue({
      connector: {
        id: 'connector-1',
        project_id: 'project-1',
        type: 'github',
        name: 'GitHub Backend',
        status: 'active',
        config: {
          type: 'github',
          base_url: 'https://api.github.com',
          project_ref: 'acme/backend',
          poll_interval: '5m',
          sync_direction: 'bidirectional',
          filters: { labels: ['openase'], exclude_labels: [], states: [], authors: [] },
          status_mapping: { open: 'Todo', closed: 'Done' },
          auto_workflow: '',
          auth_token_configured: true,
          webhook_secret_configured: true,
        },
        last_sync_at: '2026-03-31T09:05:00Z',
        last_error: '',
        stats: { total_synced: 5, synced24h: 3, failed_count: 0 },
      },
      report: {
        connectors_scanned: 1,
        connectors_synced: 1,
        connectors_failed: 0,
        issues_synced: 1,
      },
    })

    const { findByText, findByDisplayValue } = render(ConnectorsSettings)

    expect(capabilityCatalog.connectorsSettings.state).toBe('available')
    expect(await findByText('Project connectors')).toBeTruthy()
    expect(await findByText('GitHub Backend')).toBeTruthy()
    expect(await findByText('Sync now')).toBeTruthy()

    await fireEvent.click(await findByText('Edit'))

    await waitFor(() => {
      expect(getIssueConnectorStats).toHaveBeenCalledWith('connector-1')
    })
    expect(await findByDisplayValue('acme/backend')).toBeTruthy()
  })

  it('creates a connector from the settings editor', async () => {
    appStore.currentProject = {
      id: 'project-1',
      organization_id: 'org-1',
      name: 'Atlas',
      slug: 'atlas',
      description: '',
      status: 'active',
      default_agent_provider_id: null,
      accessible_machine_ids: [],
      max_concurrent_agents: 4,
    }

    listIssueConnectors.mockResolvedValue({ connectors: [] })
    createIssueConnector.mockResolvedValue({
      connector: {
        id: 'connector-1',
        project_id: 'project-1',
        type: 'github',
        name: 'GitHub Backend',
        status: 'active',
        config: {
          type: 'github',
          base_url: '',
          project_ref: 'acme/backend',
          poll_interval: '5m',
          sync_direction: 'bidirectional',
          filters: { labels: ['openase'], exclude_labels: [], states: [], authors: [] },
          status_mapping: { open: 'Todo', closed: 'Done' },
          auto_workflow: '',
          auth_token_configured: true,
          webhook_secret_configured: false,
        },
        last_sync_at: null,
        last_error: '',
        stats: { total_synced: 0, synced24h: 0, failed_count: 0 },
      },
    })

    const { findByLabelText, findByText } = render(ConnectorsSettings)

    await findByText(/No connectors configured yet\./)

    await fireEvent.input(await findByLabelText('Name'), {
      target: { value: 'GitHub Backend' },
    })
    await fireEvent.input(await findByLabelText('Project ref'), {
      target: { value: 'acme/backend' },
    })
    await fireEvent.click(await findByText('Create connector'))

    await waitFor(() => {
      expect(createIssueConnector).toHaveBeenCalledWith(
        'project-1',
        expect.objectContaining({
          name: 'GitHub Backend',
          type: 'github',
        }),
      )
      expect(toastStore.success).toHaveBeenCalledWith('Created connector "GitHub Backend".')
    })
  })
})
