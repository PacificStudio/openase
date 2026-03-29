import { describe, expect, it } from 'vitest'

import {
  derivePrimaryRepositoryReadiness,
  formatMirrorTimestamp,
  projectRepoMirrorProjection,
} from './repositories-readiness'

describe('repositories readiness', () => {
  it('treats repo bindings without a primary repo as missing_primary_repo', () => {
    expect(
      derivePrimaryRepositoryReadiness([
        {
          id: 'repo-1',
          project_id: 'project-1',
          name: 'frontend',
          repository_url: 'https://github.com/acme/frontend.git',
          default_branch: 'main',
          workspace_dirname: 'frontend',
          is_primary: false,
          labels: [],
          mirror_count: null,
          mirror_state: null,
          mirror_machine_id: null,
          last_synced_at: null,
          last_verified_at: null,
          last_error: null,
        },
      ]),
    ).toEqual({
      kind: 'missing_primary_repo',
      repoCount: 1,
      action: 'bind_primary_repo',
    })
  })

  it('maps a ready primary mirror into a ready readiness projection', () => {
    expect(
      derivePrimaryRepositoryReadiness([
        {
          id: 'repo-1',
          project_id: 'project-1',
          name: 'backend',
          repository_url: 'https://github.com/acme/backend.git',
          default_branch: 'main',
          workspace_dirname: 'backend',
          is_primary: true,
          labels: [],
          mirror_count: 2,
          mirror_state: 'ready',
          mirror_machine_id: 'machine-1',
          last_synced_at: '2026-03-29T12:00:00Z',
          last_verified_at: '2026-03-29T12:05:00Z',
        },
      ] as unknown as Parameters<typeof derivePrimaryRepositoryReadiness>[0]),
    ).toMatchObject({
      kind: 'ready',
      primaryRepoId: 'repo-1',
      primaryRepoName: 'backend',
      mirrorCount: 2,
      mirrorState: 'ready',
      mirrorMachineId: 'machine-1',
      action: 'none',
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
        is_primary: true,
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
        is_primary: true,
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
