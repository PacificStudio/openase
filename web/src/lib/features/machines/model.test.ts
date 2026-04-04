import { describe, expect, it } from 'vitest'

import { createEmptyMachineDraft, parseMachineDraft } from './model'

describe('parseMachineDraft', () => {
  it('requires advertised endpoint for listener websocket machines', () => {
    const draft = {
      ...createEmptyMachineDraft(),
      name: 'listener-01',
      host: 'listener.internal',
      connectionMode: 'ws_listener' as const,
    }

    expect(parseMachineDraft(draft)).toEqual({
      ok: false,
      error: 'Advertised websocket endpoint is required for listener machines.',
    })
  })

  it('accepts valid listener websocket machine drafts without SSH credentials', () => {
    const draft = {
      ...createEmptyMachineDraft(),
      name: 'listener-01',
      host: 'listener.internal',
      connectionMode: 'ws_listener' as const,
      advertisedEndpoint: 'wss://listener.internal/openase/transport',
      sshUser: '',
      sshKeyPath: '',
    }

    expect(parseMachineDraft(draft)).toEqual({
      ok: true,
      value: expect.objectContaining({
        connection_mode: 'ws_listener',
        advertised_endpoint: 'wss://listener.internal/openase/transport',
        ssh_user: '',
        ssh_key_path: '',
      }),
    })
  })
})
