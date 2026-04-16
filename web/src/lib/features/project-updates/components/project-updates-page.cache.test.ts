import { render, waitFor } from '@testing-library/svelte'
import { describe, expect, it } from 'vitest'

import {
  listProjectUpdates,
  makeProjectUpdatesPayload,
  makeThreadRecord,
  projectFixture,
  subscribeProjectEvents,
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
    listProjectUpdates.mockResolvedValue(
      makeProjectUpdatesPayload({
        threads: [makeThreadRecord({ body_markdown: 'Sprint 2 rollout' })],
      }),
    )

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
    listProjectUpdates.mockResolvedValue(
      makeProjectUpdatesPayload({
        threads: [makeThreadRecord({ body_markdown: 'Sprint 2 rollout' })],
      }),
    )

    const firstRender = render(ProjectUpdatesPage)
    expect(await firstRender.findByText('Sprint 2 rollout')).toBeTruthy()
    firstRender.unmount()

    markProjectUpdatesCacheDirty(projectFixture.id)

    const deferredUpdates = createDeferred<ReturnType<typeof makeProjectUpdatesPayload>>()
    listProjectUpdates.mockImplementationOnce(() => deferredUpdates.promise)

    const secondRender = render(ProjectUpdatesPage)
    expect(await secondRender.findByText('Sprint 2 rollout')).toBeTruthy()
    expect(listProjectUpdates).toHaveBeenCalledTimes(2)

    deferredUpdates.resolve(
      makeProjectUpdatesPayload({
        threads: [
          makeThreadRecord({
            title: 'Sprint 2 rollout (fresh)',
            body_markdown: 'Sprint 2 rollout (fresh)',
            last_activity_at: '2026-04-01T12:00:00Z',
          }),
        ],
      }),
    )

    await waitFor(() => {
      expect(secondRender.getByText('Sprint 2 rollout (fresh)')).toBeTruthy()
    })
  })

  it('revalidates updates after reconnect even when no later stream event arrives', async () => {
    appStore.currentProject = projectFixture
    listProjectUpdates
      .mockResolvedValueOnce(
        makeProjectUpdatesPayload({
          threads: [makeThreadRecord({ body_markdown: 'Sprint 2 rollout' })],
        }),
      )
      .mockResolvedValueOnce(
        makeProjectUpdatesPayload({
          threads: [
            makeThreadRecord({
              title: 'Recovered rollout status',
              body_markdown: 'Recovered rollout status',
              last_activity_at: '2026-04-01T12:00:00Z',
            }),
          ],
        }),
      )

    const view = render(ProjectUpdatesPage)
    expect(await view.findByText('Sprint 2 rollout')).toBeTruthy()

    for (const [, , options] of subscribeProjectEvents.mock.calls) {
      ;(options as { onReconnectRecovery?: (recovery: { sequence: number }) => void } | undefined)
        ?.onReconnectRecovery?.({ sequence: 1 })
    }

    await waitFor(() => {
      expect(listProjectUpdates).toHaveBeenCalledTimes(2)
      expect(view.getByText('Recovered rollout status')).toBeTruthy()
    })
  })
})
