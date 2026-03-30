import { describe, expect, it } from 'vitest'

import {
  parseRepositoryMirrorDraft,
  suggestRepositoryMirrorLocalPath,
} from './repository-mirror-model'

describe('repository mirror model', () => {
  it('allows prepare mode to omit local_path so the backend can derive it', () => {
    const parsed = parseRepositoryMirrorDraft({
      machineId: 'machine-1',
      localPath: '',
      mode: 'prepare',
    })

    expect(parsed).toEqual({
      ok: true,
      value: {
        machine_id: 'machine-1',
        mode: 'prepare',
      },
    })
  })

  it('still requires local_path for register_existing mode', () => {
    const parsed = parseRepositoryMirrorDraft({
      machineId: 'machine-1',
      localPath: '',
      mode: 'register_existing',
    })

    expect(parsed).toEqual({
      ok: false,
      error: 'Local path is required.',
    })
  })

  it('derives a remote suggested mirror path from mirror_root and project context', () => {
    const suggestion = suggestRepositoryMirrorLocalPath(
      {
        id: 'machine-1',
        name: 'gpu-runner-01',
        host: '10.0.0.42',
        status: 'online',
        mirror_root: '/srv/openase/mirrors',
      } as never,
      {
        id: 'repo-1',
        name: 'backend',
      } as never,
      'acme',
      'openase',
    )

    expect(suggestion).toBe('/srv/openase/mirrors/acme/openase/backend')
  })
})
