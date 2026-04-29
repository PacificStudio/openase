import { cleanup, render, waitFor } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { i18nStore } from '$lib/i18n/store.svelte'
import { appStore } from '$lib/stores/app.svelte'
import { markAgentsPageCacheDirty, resetAgentsPageCacheForTests } from '../agents-page-cache'
import AgentsPage from './agents-page.svelte'
import { makeAgent, makePageData, orgFixture, projectFixture } from './agents-page.test-helpers'

const { connectEventStream, loadAgentsPageResult, subscribeProjectEvents } = vi.hoisted(() => ({
  connectEventStream: vi.fn(),
  loadAgentsPageResult: vi.fn(),
  subscribeProjectEvents: vi.fn(),
}))

vi.mock('$lib/api/openase', () => ({}))

vi.mock('$lib/api/sse', () => ({
  connectEventStream,
}))

vi.mock('$lib/features/project-events', () => ({
  createProjectReconnectRecoveryTask: vi.fn(
    (recover: (recovery: { sequence: number }) => Promise<void> | void) =>
      (recovery: { sequence: number }) => {
        void recover(recovery)
      },
  ),
  isProjectDashboardRefreshEvent: vi.fn(() => false),
  readProjectDashboardRefreshSections: vi.fn(() => []),
  subscribeProjectEvents,
}))

vi.mock('../page-data', () => ({
  loadAgentsPageResult,
}))

describe('AgentsPage', () => {
  afterEach(() => {
    cleanup()
    resetAgentsPageCacheForTests()
    i18nStore.setLocale('en')
    appStore.currentOrg = null
    appStore.currentProject = null
    vi.clearAllMocks()
  })

  it('renders agent list after loading', async () => {
    appStore.currentOrg = orgFixture
    appStore.currentProject = projectFixture

    loadAgentsPageResult.mockResolvedValue({ ok: true, data: makePageData(makeAgent()) })
    connectEventStream.mockReturnValue(() => {})
    subscribeProjectEvents.mockReturnValue(() => {})

    const { findByText } = render(AgentsPage)

    expect(await findByText('Codex Worker')).toBeTruthy()
  })

  it('renders localized page chrome in Chinese', async () => {
    i18nStore.setLocale('zh')
    appStore.currentOrg = orgFixture
    appStore.currentProject = projectFixture

    loadAgentsPageResult.mockResolvedValue({ ok: true, data: makePageData(makeAgent()) })
    connectEventStream.mockReturnValue(() => {})
    subscribeProjectEvents.mockReturnValue(() => {})

    const { findByText } = render(AgentsPage)

    expect(await findByText('代理')).toBeTruthy()
    expect(await findByText('管理代理定义并跟踪各自的运行状态。')).toBeTruthy()
    expect(await findByText('注册代理')).toBeTruthy()
    expect(await findByText('运行中 1/1')).toBeTruthy()
  })

  it('reuses the cached agents page snapshot when remounting in the same project', async () => {
    appStore.currentOrg = orgFixture
    appStore.currentProject = projectFixture

    loadAgentsPageResult.mockResolvedValue({ ok: true, data: makePageData(makeAgent()) })
    connectEventStream.mockReturnValue(() => {})
    subscribeProjectEvents.mockReturnValue(() => {})

    const firstRender = render(AgentsPage)
    expect(await firstRender.findByText('Codex Worker')).toBeTruthy()
    expect(loadAgentsPageResult).toHaveBeenCalledTimes(1)

    firstRender.unmount()

    const secondRender = render(AgentsPage)
    expect(await secondRender.findByText('Codex Worker')).toBeTruthy()
    expect(loadAgentsPageResult).toHaveBeenCalledTimes(1)
  })

  it('shows cached agents immediately and refreshes in the background when the cache is dirty', async () => {
    appStore.currentOrg = orgFixture
    appStore.currentProject = projectFixture

    loadAgentsPageResult.mockResolvedValue({ ok: true, data: makePageData(makeAgent()) })
    connectEventStream.mockReturnValue(() => {})
    subscribeProjectEvents.mockReturnValue(() => {})

    const firstRender = render(AgentsPage)
    expect(await firstRender.findByText('Codex Worker')).toBeTruthy()
    firstRender.unmount()

    markAgentsPageCacheDirty('project-1', 'org-1')

    const deferred = createDeferred<{ ok: true; data: ReturnType<typeof makePageData> }>()
    loadAgentsPageResult.mockImplementationOnce(() => deferred.promise)

    const secondRender = render(AgentsPage)
    expect(await secondRender.findByText('Codex Worker')).toBeTruthy()
    expect(loadAgentsPageResult).toHaveBeenCalledTimes(2)

    deferred.resolve({ ok: true, data: makePageData(makeAgent({ name: 'Claude Worker' })) })

    await waitFor(() => {
      expect(secondRender.getByText('Claude Worker')).toBeTruthy()
    })
  })
})

function createDeferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((res, rej) => {
    resolve = res
    reject = rej
  })

  return { promise, resolve, reject }
}
