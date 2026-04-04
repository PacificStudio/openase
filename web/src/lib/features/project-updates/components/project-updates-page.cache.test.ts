import { render, waitFor } from '@testing-library/svelte'
import { describe, expect, it } from 'vitest'

import {
  listProjectUpdates,
  makeThreadRecord,
  projectFixture,
  setupProjectUpdatesPageTest,
} from './project-updates-page.test-support'
import { ProjectUpdatesPage } from '..'
import { markProjectUpdatesCacheDirty } from '../project-updates-cache'
import { appStore } from '$lib/stores/app.svelte'

function createDeferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((nextResolve, nextReject) => {
    resolve = nextResolve
    reject = nextReject
  })
  return { promise, resolve, reject }
}

describe('ProjectUpdatesPage cache behavior', () => {
  setupProjectUpdatesPageTest()

  it('reuses the cached updates snapshot when remounting the page in the same project', async () => {
    listProjectUpdates.mockResolvedValue({ threads: [makeThreadRecord()] })

    const firstRender = render(ProjectUpdatesPage)
    expect(await firstRender.findByText('Sprint 2 rollout')).toBeTruthy()
    expect(listProjectUpdates).toHaveBeenCalledTimes(1)

    firstRender.unmount()

    const secondRender = render(ProjectUpdatesPage)
    expect(await secondRender.findByText('Sprint 2 rollout')).toBeTruthy()
    expect(listProjectUpdates).toHaveBeenCalledTimes(1)
  })

  it('shows cached updates immediately and refreshes in the background once the cache is dirty', async () => {
    appStore.currentProject = projectFixture
    listProjectUpdates.mockResolvedValue({ threads: [makeThreadRecord()] })

    const firstRender = render(ProjectUpdatesPage)
    expect(await firstRender.findByText('Sprint 2 rollout')).toBeTruthy()
    firstRender.unmount()

    markProjectUpdatesCacheDirty(projectFixture.id)

    const deferredUpdates = createDeferred<{ threads: ReturnType<typeof makeThreadRecord>[] }>()
    listProjectUpdates.mockImplementationOnce(() => deferredUpdates.promise)

    const secondRender = render(ProjectUpdatesPage)
    expect(await secondRender.findByText('Sprint 2 rollout')).toBeTruthy()
    expect(listProjectUpdates).toHaveBeenCalledTimes(2)

    deferredUpdates.resolve({
      threads: [
        makeThreadRecord({
          title: 'Sprint 2 rollout (fresh)',
          last_activity_at: '2026-04-01T12:00:00Z',
        }),
      ],
    })

    await waitFor(() => {
      expect(secondRender.getByText('Sprint 2 rollout (fresh)')).toBeTruthy()
    })
  })
})
