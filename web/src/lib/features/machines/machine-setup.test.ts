import { describe, expect, it } from 'vitest'

import type { Machine } from '$lib/api/contracts'
import { buildMachineSetupGuide, friendlyTransportLabel } from './machine-setup'

function machineFixture(overrides: Partial<Machine> = {}): Machine {
  return {
    id: 'machine-1',
    organization_id: 'org-1',
    name: 'GPU Builder',
    host: 'builder.internal',
    port: 22,
    reachability_mode: 'direct_connect',
    execution_mode: 'ssh_compat',
    execution_capabilities: ['probe'],
    ssh_helper_enabled: true,
    ssh_helper_required: true,
    connection_mode: 'ssh',
    transport_capabilities: ['probe'],
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

describe('machine setup guidance', () => {
  it('surfaces legacy ssh migration and helper quick setup for direct-connect records', () => {
    const guide = buildMachineSetupGuide({
      machine: machineFixture(),
    })

    expect(guide.runtimeLabel).toBe('Legacy SSH runtime')
    expect(guide.helperLabel).toBe('SSH helper required')
    expect(guide.stateLabel).toBe('Migration needed')
    expect(guide.nextSteps).toContain('Expose the machine listener endpoint and save it here.')
    expect(guide.commands).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          title: 'SSH quick setup',
          command: 'ssh -i /keys/id_ed25519 ubuntu@builder.internal',
        }),
        expect.objectContaining({
          title: 'Control-plane connection test',
          command: 'openase machine test machine-1',
        }),
      ]),
    )
  })

  it('builds a self-serve reverse-connect command path and optional ssh bootstrap', () => {
    const guide = buildMachineSetupGuide({
      machine: machineFixture({
        reachability_mode: 'reverse_connect',
        execution_mode: 'websocket',
        connection_mode: 'ws_reverse',
        ssh_helper_required: false,
        daemon_status: {
          registered: false,
          last_registered_at: null,
          current_session_id: null,
          session_state: 'unknown',
        },
      }),
    })

    expect(guide.topologyLabel).toBe('Machine dials out to OpenASE')
    expect(guide.runtimeLabel).toBe('Reverse-connect daemon')
    expect(guide.stateLabel).toBe('Waiting for daemon registration')
    expect(guide.commands).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          title: '1. Issue a machine channel token on the control plane',
          command: 'openase machine issue-channel-token --machine-id machine-1 --format shell',
        }),
        expect.objectContaining({
          title: '2. Paste those exports on the remote machine and start the daemon',
          command: 'openase machine-agent run',
        }),
        expect.objectContaining({
          title: 'Optional SSH bootstrap',
          command: 'openase machine ssh-bootstrap machine-1',
        }),
        expect.objectContaining({
          title: 'SSH diagnostics',
          command: 'openase machine ssh-diagnostics machine-1',
        }),
      ]),
    )
  })

  it('maps raw transport names to user-facing runtime labels', () => {
    expect(friendlyTransportLabel('ws_listener')).toBe('Direct-connect listener')
    expect(friendlyTransportLabel('ws_reverse')).toBe('Reverse-connect daemon')
    expect(friendlyTransportLabel('ssh')).toBe('SSH helper path')
  })
})
