import { ApiError } from '$lib/api/client'
import type { TicketDetailLiveContext, TicketDetailProjectReferenceData } from './context'
import { selectTicketDetailReferenceData } from './context'
import type {
  TicketDetail,
  TicketReferenceOption,
  TicketRepoOption,
  TicketStatusOption,
} from './types'

type SelectedReferenceData = {
  statuses: TicketStatusOption[]
  dependencyCandidates: TicketReferenceOption[]
  repoOptions: TicketRepoOption[]
}

type ReferenceControllerInput = {
  fetchLiveContext(
    projectId: string,
    ticketId: string,
    refs: TicketDetailProjectReferenceData,
  ): Promise<TicketDetailLiveContext>
  fetchReferenceData(projectId: string): Promise<TicketDetailProjectReferenceData>
  getLoadRequestId(): number
  isLoading(): boolean
  getTicket(): TicketDetail | null
  applyReferenceSelection(selected: SelectedReferenceData): void
  applyTimelineRefresh(detailContext: TicketDetailLiveContext): void
  notifyError(message: string): void
}

export function createTicketDrawerReferenceController(input: ReferenceControllerInput) {
  let referenceData: TicketDetailProjectReferenceData | null = null
  let referenceProjectId: string | null = null
  let timelineRefreshQueued = false
  let timelineRefreshLoop: Promise<void> | null = null
  let referenceRefreshQueued = false
  let referenceRefreshLoop: Promise<void> | null = null

  const hasCachedReferenceData = (projectId: string) =>
    referenceProjectId === projectId && referenceData !== null
  const activeTicketChanged = (ticketId: string) => input.getTicket()?.id !== ticketId

  async function ensureReferenceData(projectId: string) {
    if (hasCachedReferenceData(projectId)) {
      return referenceData
    }

    const nextReferenceData = await input.fetchReferenceData(projectId)
    referenceProjectId = projectId
    referenceData = nextReferenceData
    return nextReferenceData
  }

  function applyReferenceData(
    projectId: string,
    ticketId: string,
    nextReferenceData: TicketDetailProjectReferenceData,
  ) {
    referenceProjectId = projectId
    referenceData = nextReferenceData
    input.applyReferenceSelection(selectTicketDetailReferenceData(nextReferenceData, ticketId))
  }

  async function runTimelineRefresh(projectId: string, ticketId: string) {
    if (input.isLoading() || !input.getTicket()) {
      return
    }
    if (timelineRefreshLoop) {
      await timelineRefreshLoop
      return
    }

    timelineRefreshLoop = (async () => {
      while (timelineRefreshQueued && !input.isLoading() && input.getTicket()) {
        timelineRefreshQueued = false
        const requestId = input.getLoadRequestId()
        try {
          const currentReferenceData = await ensureReferenceData(projectId)
          if (requestId !== input.getLoadRequestId() || !input.getTicket()) {
            continue
          }
          const detailContext = await input.fetchLiveContext(
            projectId,
            ticketId,
            currentReferenceData!,
          )
          if (requestId !== input.getLoadRequestId() || !input.getTicket()) {
            continue
          }
          input.applyTimelineRefresh(detailContext)
        } catch (caughtError) {
          if (requestId !== input.getLoadRequestId() || !input.getTicket()) {
            continue
          }
          input.notifyError(
            caughtError instanceof ApiError
              ? caughtError.detail
              : 'Failed to refresh ticket timeline.',
          )
        }
      }
    })().finally(() => {
      timelineRefreshLoop = null
    })

    await timelineRefreshLoop
  }

  async function runReferenceRefresh(projectId: string, ticketId: string) {
    if (input.isLoading()) {
      return
    }
    if (referenceRefreshLoop) {
      await referenceRefreshLoop
      return
    }

    referenceRefreshLoop = (async () => {
      while (referenceRefreshQueued && !input.isLoading()) {
        referenceRefreshQueued = false
        const requestId = input.getLoadRequestId()
        try {
          const nextReferenceData = await input.fetchReferenceData(projectId)
          if (requestId !== input.getLoadRequestId() || activeTicketChanged(ticketId)) {
            continue
          }
          applyReferenceData(projectId, ticketId, nextReferenceData)
        } catch (caughtError) {
          if (requestId !== input.getLoadRequestId() || activeTicketChanged(ticketId)) {
            continue
          }
          input.notifyError(
            caughtError instanceof ApiError
              ? caughtError.detail
              : 'Failed to refresh ticket references.',
          )
        }
      }
    })().finally(() => {
      referenceRefreshLoop = null
    })

    await referenceRefreshLoop
  }

  return {
    hasCachedReferenceData,
    ensureReferenceData,
    applyReferenceData,
    async refreshTimeline(projectId: string, ticketId: string) {
      if (input.isLoading() || !input.getTicket()) {
        return
      }
      timelineRefreshQueued = true
      await runTimelineRefresh(projectId, ticketId)
      if (timelineRefreshQueued) {
        await runTimelineRefresh(projectId, ticketId)
      }
    },
    async refreshReferences(projectId: string, ticketId: string) {
      if (input.isLoading()) {
        return
      }
      referenceRefreshQueued = true
      await runReferenceRefresh(projectId, ticketId)
      if (referenceRefreshQueued) {
        await runReferenceRefresh(projectId, ticketId)
      }
    },
    resetQueues() {
      timelineRefreshQueued = false
      timelineRefreshLoop = null
      referenceRefreshQueued = false
      referenceRefreshLoop = null
    },
    invalidateReferences(projectId?: string) {
      if (!projectId || referenceProjectId === projectId) {
        referenceProjectId = null
        referenceData = null
      }
    },
  }
}
