import { describe, expect, it } from 'vitest'

import type { Machine } from '$lib/api/contracts'
import {
  createEmptyMachineDraft,
  machineToDraft,
  parseMachineDraft,
  updateMachineDraft,
} from './model'

function machineFixture(overrides: Partial<Machine> = {}): Machine {
  return {
    id: 'machine-1',
    organization_id: 'org-1',
    name: 'GPU Builder',
    host: 'builder.internal',
    port: 22,
    reachability_mode: 'direct_connect',
    execution_mode: 'websocket',
    execution_capabilities: ['probe'],
    ssh_helper_enabled: true,
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
    labels: [],
    status: 'online',
    workspace_root: '/home/ubuntu/.openase/workspace',
    agent_cli_path: '/usr/local/bin/openase-agent',
    env_vars: [],
    resources: {},
    last_heartbeat_at: null,
    created_at: '2026-04-02T09:00:00Z',
    updated_at: '2026-04-02T10:00:00Z',
    ...overrides,
  } as Machine
}

describe('machines model', () => {
  it('starts new remote drafts with the conservative workspace recommendation', () => {
    const draft = createEmptyMachineDraft()

    expect(draft.reachabilityMode).toBe('direct_connect')
    expect(draft.executionMode).toBe('websocket')
    expect(draft.workspaceRoot).toBe('/srv/openase/workspace')
  })

  it('switches local drafts to the reserved local identity and workspace convention', () => {
    const draft = updateMachineDraft(createEmptyMachineDraft(), 'reachabilityMode', 'local', null)

    expect(draft.reachabilityMode).toBe('local')
    expect(draft.executionMode).toBe('local_process')
    expect(draft.name).toBe('local')
    expect(draft.host).toBe('local')
    expect(draft.workspaceRoot).toBe('~/.openase/workspace')
  })

  it('refreshes the recommended Linux workspace root when the SSH user changes', () => {
    const machine = machineFixture()
    const draft = machineToDraft(machine)

    const nextDraft = updateMachineDraft(draft, 'sshUser', 'deploy', machine)

    expect(nextDraft.workspaceRoot).toBe('/home/deploy/.openase/workspace')
  })

  it('accepts direct-connect websocket drafts when a listener endpoint is provided', () => {
    const draft = machineToDraft(
      machineFixture({
        advertised_endpoint: 'wss://builder.internal/openase/transport',
      }),
    )

    const parsed = parseMachineDraft(draft)

    expect(parsed.ok).toBe(true)
    if (parsed.ok) {
      expect(parsed.value.execution_mode).toBe('websocket')
      expect(parsed.value.advertised_endpoint).toBe('wss://builder.internal/openase/transport')
    }
  })

  it('requires direct-connect websocket drafts to include a listener endpoint even with SSH helper credentials', () => {
    const parsed = parseMachineDraft(machineToDraft(machineFixture()))

    expect(parsed.ok).toBe(false)
    if (!parsed.ok) {
      expect(parsed.error).toContain('Advertised endpoint is required')
    }
  })

  it('requires an advertised endpoint for websocket listener machines', () => {
    const draft = createEmptyMachineDraft()
    draft.name = 'listener'
    draft.host = 'listener.internal'
    draft.port = '443'
    draft.workspaceRoot = '/srv/openase/workspace'

    const parsed = parseMachineDraft(draft)
    expect(parsed.ok).toBe(false)
    if (!parsed.ok) {
      expect(parsed.error).toContain('Advertised endpoint is required')
    }

    const withEndpoint = updateMachineDraft(
      draft,
      'advertisedEndpoint',
      'wss://listener.internal/openase',
      null,
    )
    const parsedWithEndpoint = parseMachineDraft(withEndpoint)
    expect(parsedWithEndpoint.ok).toBe(true)
  })

  it('rejects listener endpoints that do not use websocket schemes', () => {
    const draft = createEmptyMachineDraft()
    draft.name = 'listener'
    draft.host = 'listener.internal'
    draft.port = '443'
    draft.workspaceRoot = '/srv/openase/workspace'
    draft.advertisedEndpoint = 'https://listener.internal/openase'

    expect(parseMachineDraft(draft)).toEqual({
      ok: false,
      error: 'Advertised endpoint must use ws:// or wss://.',
    })
  })

  it('accepts valid listener websocket machine drafts without SSH credentials', () => {
    const draft = {
      ...createEmptyMachineDraft(),
      name: 'listener-01',
      host: 'listener.internal',
      reachabilityMode: 'direct_connect' as const,
      executionMode: 'websocket' as const,
      advertisedEndpoint: 'wss://listener.internal/openase/transport',
      sshUser: '',
      sshKeyPath: '',
      workspaceRoot: '/srv/openase/workspace',
    }

    expect(parseMachineDraft(draft)).toEqual({
      ok: true,
      value: expect.objectContaining({
        reachability_mode: 'direct_connect',
        execution_mode: 'websocket',
        advertised_endpoint: 'wss://listener.internal/openase/transport',
        ssh_user: '',
        ssh_key_path: '',
      }),
    })
  })
})
