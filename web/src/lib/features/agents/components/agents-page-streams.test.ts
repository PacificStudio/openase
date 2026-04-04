import { afterEach, describe, expect, it, vi } from 'vitest'
import { connectAgentsPageStreams } from './agents-page-streams'

const {
  connectEventStream,
  isProjectDashboardRefreshEvent,
  readProjectDashboardRefreshSections,
  subscribeProjectEvents,
} = vi.hoisted(() => ({
  connectEventStream: vi.fn(),
  isProjectDashboardRefreshEvent: vi.fn(),
  readProjectDashboardRefreshSections: vi.fn(),
  subscribeProjectEvents: vi.fn(),
}))

vi.mock('$lib/api/sse', () => ({
  connectEventStream,
}))

vi.mock('$lib/features/project-events', () => ({
  isProjectDashboardRefreshEvent,
  readProjectDashboardRefreshSections,
  subscribeProjectEvents,
}))

describe('connectAgentsPageStreams', () => {
  afterEach(() => {
    vi.clearAllMocks()
  })

  it('reloads on relevant project dashboard refresh sections and provider stream events', () => {
    let projectListener: ((event: unknown) => void) | null = null
    let providerListener: (() => void) | null = null
    const onEvent = vi.fn()

    subscribeProjectEvents.mockImplementation((_projectId, listener) => {
      projectListener = listener
      return () => {}
    })
    connectEventStream.mockImplementation((_path, options) => {
      providerListener = options.onEvent
      return () => {}
    })
    isProjectDashboardRefreshEvent.mockReturnValue(true)
    readProjectDashboardRefreshSections.mockReturnValue(['agents'])

    connectAgentsPageStreams('project-1', 'org-1', onEvent)

    expect(projectListener).not.toBeNull()
    expect(providerListener).not.toBeNull()
    projectListener!({ topic: 'project.dashboard.events', type: 'project.dashboard.refresh' })
    providerListener!()

    expect(onEvent).toHaveBeenCalledTimes(2)
  })

  it('ignores unrelated project events', () => {
    let projectListener: ((event: unknown) => void) | null = null
    const onEvent = vi.fn()

    subscribeProjectEvents.mockImplementation((_projectId, listener) => {
      projectListener = listener
      return () => {}
    })
    connectEventStream.mockReturnValue(() => {})
    isProjectDashboardRefreshEvent.mockReturnValue(false)

    connectAgentsPageStreams('project-1', 'org-1', onEvent)

    expect(projectListener).not.toBeNull()
    projectListener!({ topic: 'agent.events', type: 'agent.heartbeat' })

    expect(onEvent).not.toHaveBeenCalled()
  })

  it('reloads immediately on non-heartbeat agent lifecycle events', () => {
    let projectListener: ((event: unknown) => void) | null = null
    const onEvent = vi.fn()

    subscribeProjectEvents.mockImplementation((_projectId, listener) => {
      projectListener = listener
      return () => {}
    })
    connectEventStream.mockReturnValue(() => {})

    connectAgentsPageStreams('project-1', 'org-1', onEvent)

    expect(projectListener).not.toBeNull()

    projectListener!({ topic: 'agent.events', type: 'agent.executing' })
    projectListener!({ topic: 'agent.events', type: 'agent.heartbeat' })

    expect(onEvent).toHaveBeenCalledTimes(1)
  })

  it('reloads immediately on ticket run lifecycle events', () => {
    let projectListener: ((event: unknown) => void) | null = null
    const onEvent = vi.fn()

    subscribeProjectEvents.mockImplementation((_projectId, listener) => {
      projectListener = listener
      return () => {}
    })
    connectEventStream.mockReturnValue(() => {})

    connectAgentsPageStreams('project-1', 'org-1', onEvent)

    expect(projectListener).not.toBeNull()

    projectListener!({ topic: 'ticket.run.events', type: 'ticket.run.lifecycle' })
    projectListener!({ topic: 'ticket.run.events', type: 'ticket.run.trace' })

    expect(onEvent).toHaveBeenCalledTimes(1)
  })
})
