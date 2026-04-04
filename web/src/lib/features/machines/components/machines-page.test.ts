import { cleanup, fireEvent, render, waitFor, within } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { MachineItem, MachineSnapshot } from '../types'
import { appStore } from '$lib/stores/app.svelte'
import MachinesPage from './machines-page.svelte'
import { markMachinesPageCacheDirty, resetMachinesPageCacheForTests } from '../machines-page-cache'
import { resetOrganizationEventBusForTests } from '$lib/features/org-events'

const {
  loadMachines,
  loadMachineSnapshot,
  machineErrorMessage,
  removeMachine,
  runMachineConnectionTest,
  runMachineHealthRefresh,
  saveMachine,
  subscribeOrganizationMachineEvents,
} = vi.hoisted(() => ({
  loadMachines: vi.fn(),
  loadMachineSnapshot: vi.fn(),
  machineErrorMessage: vi.fn((_: unknown, fallback: string) => fallback),
  removeMachine: vi.fn(),
  runMachineConnectionTest: vi.fn(),
  runMachineHealthRefresh: vi.fn(),
  saveMachine: vi.fn(),
  subscribeOrganizationMachineEvents: vi.fn(),
}))

vi.mock('./machines-page-api', () => ({
  loadMachines,
  loadMachineSnapshot,
  machineErrorMessage,
  removeMachine,
  runMachineConnectionTest,
  runMachineHealthRefresh,
  saveMachine,
}))

vi.mock('$lib/features/org-events', async () => {
  const actual = await vi.importActual<typeof import('$lib/features/org-events')>(
    '$lib/features/org-events',
  )
  return {
    ...actual,
    subscribeOrganizationMachineEvents,
  }
})

vi.mock('./machines-page-focus', () => ({
  syncMachinesPageProjectAIFocus: vi.fn(() => () => {}),
}))

function createDeferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((nextResolve, nextReject) => {
    resolve = nextResolve
    reject = nextReject
  })
  return { promise, resolve, reject }
}

const machineFixture = {
  id: 'machine-1',
  organization_id: 'org-1',
  name: 'GPU Builder',
  host: 'builder.internal',
  port: 22,
  connection_mode: 'ssh',
  transport_capabilities: ['probe', 'workspace_prepare', 'artifact_sync', 'process_streaming'],
  ssh_user: 'ubuntu',
  ssh_key_path: '/keys/id_ed25519',
  advertised_endpoint: null,
  daemon_status: {
    registered: false,
    last_registered_at: null,
    current_session_id: null,
    session_state: 'unknown',
  },
  detected_os: 'linux',
  detected_arch: 'amd64',
  detection_status: 'ok',
  detection_message: 'Detected amd64 on Linux.',
  channel_credential: {
    kind: 'none',
    token_id: null,
    certificate_id: null,
  },
  description: '',
  labels: ['gpu'],
  status: 'online',
  workspace_root: '/workspace',
  agent_cli_path: '/usr/local/bin/openase-agent',
  env_vars: [],
  resources: {
    checked_at: '2026-04-02T10:00:00Z',
    cpu_usage_percent: 14.5,
    memory_available_gb: 43,
    monitor: {
      l4: {
        checked_at: '2026-04-02T10:00:00Z',
        agent_dispatchable: true,
      },
    },
    agent_environment: {},
  },
  last_heartbeat_at: '2026-04-02T10:00:00Z',
  created_at: '2026-04-02T09:00:00Z',
  updated_at: '2026-04-02T10:00:00Z',
} as MachineItem

const snapshotFixture: MachineSnapshot = {
  checkedAt: '2026-04-02T10:00:00Z',
  gpus: [],
  agentEnvironment: [],
  monitor: {
    l4: {
      checkedAt: '2026-04-02T10:00:00Z',
      agentDispatchable: true,
    },
  },
  monitorErrors: [],
}

describe('MachinesPage cache behavior', () => {
  beforeEach(() => {
    appStore.currentOrg = {
      id: 'org-1',
      name: 'Acme',
      slug: 'acme',
      default_agent_provider_id: null,
      status: 'active',
    }
    appStore.currentProject = {
      id: 'project-1',
      organization_id: 'org-1',
      name: 'Project One',
      slug: 'project-one',
      description: '',
      status: 'active',
      default_agent_provider_id: null,
      max_concurrent_agents: 1,
      accessible_machine_ids: [],
    }
    loadMachines.mockResolvedValue([machineFixture])
    loadMachineSnapshot.mockResolvedValue(snapshotFixture)
    subscribeOrganizationMachineEvents.mockReturnValue(() => {})
  })

  afterEach(() => {
    cleanup()
    resetMachinesPageCacheForTests()
    resetOrganizationEventBusForTests()
    appStore.currentOrg = null
    appStore.currentProject = null
    vi.clearAllMocks()
  })

  it('reuses the cached machine list and selected resource snapshot when remounting in the same org', async () => {
    const firstRender = render(MachinesPage)
    expect(await firstRender.findByTestId('machine-card-machine-1')).toBeTruthy()

    await openMachineDetails('machine-1')
    expect(await firstRender.findByText('Health snapshot')).toBeTruthy()

    expect(loadMachines).toHaveBeenCalledTimes(1)
    expect(loadMachineSnapshot).toHaveBeenCalledTimes(1)

    firstRender.unmount()

    const secondRender = render(MachinesPage)
    expect(await secondRender.findByTestId('machine-card-machine-1')).toBeTruthy()
    expect(await secondRender.findByText('Health snapshot')).toBeTruthy()

    expect(loadMachines).toHaveBeenCalledTimes(1)
    expect(loadMachineSnapshot).toHaveBeenCalledTimes(1)
  })

  it('shows cached machines and resources immediately and refreshes the list in the background when the cache is dirty', async () => {
    const firstRender = render(MachinesPage)
    expect(await firstRender.findByTestId('machine-card-machine-1')).toBeTruthy()

    await openMachineDetails('machine-1')
    expect(await firstRender.findByText('Health snapshot')).toBeTruthy()
    firstRender.unmount()

    markMachinesPageCacheDirty('org-1')

    const deferredMachines = createDeferred<MachineItem[]>()
    const deferredSnapshot = createDeferred<MachineSnapshot | null>()
    loadMachines.mockImplementationOnce(() => deferredMachines.promise)
    loadMachineSnapshot.mockImplementationOnce(() => deferredSnapshot.promise)

    const secondRender = render(MachinesPage)
    expect(await secondRender.findByTestId('machine-card-machine-1')).toBeTruthy()
    expect(await secondRender.findByText('Health snapshot')).toBeTruthy()

    expect(loadMachines).toHaveBeenCalledTimes(2)
    expect(loadMachineSnapshot).toHaveBeenCalledTimes(1)

    deferredMachines.resolve([machineFixture])
    deferredSnapshot.resolve(snapshotFixture)

    await waitFor(() => {
      expect(loadMachines).toHaveBeenCalledTimes(2)
    })
  })

  it('shows connection mode, detection status, and workspace guidance in the machine editor', async () => {
    const view = render(MachinesPage)

    expect(await view.findByText('Linux / amd64')).toBeTruthy()
    expect(view.getByText('SSH')).toBeTruthy()
    expect(view.getByText('Detected')).toBeTruthy()

    await openMachineDetails('machine-1')

    expect(await view.findByText('Connection mode')).toBeTruthy()
    expect(view.getAllByText('Detected amd64 on Linux.').length).toBeGreaterThan(0)
    expect(view.getByText('Recommended root')).toBeTruthy()
    expect(view.getByText('/home/ubuntu/.openase/workspace')).toBeTruthy()
    expect(view.getByText('Keeping the saved workspace root override.')).toBeTruthy()
  })
})

async function openMachineDetails(machineId: string) {
  const card = document.querySelector(`[data-testid="machine-card-${machineId}"]`)
  if (!card) {
    throw new Error(`machine card not found for ${machineId}`)
  }

  await fireEvent.click(within(card as HTMLElement).getByTitle('View details'))
}
