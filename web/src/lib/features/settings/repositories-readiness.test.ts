import { describe, expect, it } from 'vitest'

import {
  deriveRepositoryBindingsReadiness,
  formatMirrorTimestamp,
  projectRepoMirrorProjection,
} from './repositories-readiness'

describe('repositories readiness', () => {
  it('reports missing_repo only when no repositories are configured', () => {
    expect(deriveRepositoryBindingsReadiness([])).toEqual({
      kind: 'missing_repo',
      repoCount: 0,
      action: 'add_repo',
    })
  })

  it('reports ready once at least one repository binding exists', () => {
    expect(
      deriveRepositoryBindingsReadiness([
        {
          id: 'repo-1',
          project_id: 'project-1',
          name: 'backend',
          repository_url: 'https://github.com/acme/backend.git',
          default_branch: 'main',
          workspace_dirname: 'backend',
          labels: [],
          mirror_count: 2,
          mirror_state: 'ready',
          mirror_machine_id: 'machine-1',
          last_synced_at: '2026-03-29T12:00:00Z',
          last_verified_at: '2026-03-29T12:05:00Z',
        },
      ] as unknown as Parameters<typeof deriveRepositoryBindingsReadiness>[0]),
    ).toEqual({
      kind: 'ready',
      repoCount: 1,
    })
  })

  it('maps missing mirror data into a prepare_mirror action', () => {
    expect(
      projectRepoMirrorProjection({
        id: 'repo-1',
        project_id: 'project-1',
        name: 'backend',
        repository_url: 'https://github.com/acme/backend.git',
        default_branch: 'main',
        workspace_dirname: 'backend',
        labels: [],
      } as unknown as Parameters<typeof projectRepoMirrorProjection>[0]),
    ).toEqual({
      mirrorCount: 0,
      mirrorState: 'missing',
      mirrorMachineId: null,
      lastSyncedAt: null,
      lastVerifiedAt: null,
      lastError: null,
      action: 'prepare_mirror',
    })
  })

  it('maps stale mirrors into a sync_mirror action', () => {
    expect(
      projectRepoMirrorProjection({
        id: 'repo-1',
        project_id: 'project-1',
        name: 'backend',
        repository_url: 'https://github.com/acme/backend.git',
        default_branch: 'main',
        workspace_dirname: 'backend',
        labels: [],
        mirror_count: 1,
        mirror_state: 'stale',
        mirror_machine_id: 'machine-1',
        last_error: 'fetch failed',
      } as unknown as Parameters<typeof projectRepoMirrorProjection>[0]),
    ).toMatchObject({
      mirrorCount: 1,
      mirrorState: 'stale',
      mirrorMachineId: 'machine-1',
      lastError: 'fetch failed',
      action: 'sync_mirror',
    })
  })

  it('formats mirror timestamps in a stable UTC display string', () => {
    expect(formatMirrorTimestamp('2026-03-29T12:05:00Z')).toBe('2026-03-29 12:05:00 UTC')
  })
})
