import { ApiError } from '$lib/api/client'
import type { AgentInstance } from './types'

type LoadData = (params: {
  projectId: string
  orgId: string
  showLoading: boolean
}) => Promise<void>

type RuntimeControlOptions = {
  agent: AgentInstance
  projectId?: string
  orgId?: string
  action: (agentId: string) => Promise<unknown>
  loadData: LoadData
  successMessage: string
  failureMessage: string
  setPendingAgentId: (value: string | null) => void
  setPageError: (value: string) => void
  setPageFeedback: (value: string) => void
}

type RuntimeControlHandlerOptions = {
  action: (agentId: string) => Promise<unknown>
  successPrefix: string
  failureMessage: string
  getProjectId: () => string | undefined
  getOrgId: () => string | undefined
  loadData: LoadData
  setPendingAgentId: (value: string | null) => void
  setPageError: (value: string) => void
  setPageFeedback: (value: string) => void
}

export async function runAgentRuntimeControl({
  agent,
  projectId,
  orgId,
  action,
  loadData,
  successMessage,
  failureMessage,
  setPendingAgentId,
  setPageError,
  setPageFeedback,
}: RuntimeControlOptions) {
  if (!projectId || !orgId) {
    setPageError('Project context is unavailable.')
    return
  }

  setPendingAgentId(agent.id)
  setPageError('')
  setPageFeedback('')

  try {
    await action(agent.id)
    setPageFeedback(successMessage)
    await loadData({ projectId, orgId, showLoading: false })
  } catch (caughtError) {
    setPageError(caughtError instanceof ApiError ? caughtError.detail : failureMessage)
  } finally {
    setPendingAgentId(null)
  }
}

export function createRuntimeControlHandler({
  action,
  successPrefix,
  failureMessage,
  getProjectId,
  getOrgId,
  loadData,
  setPendingAgentId,
  setPageError,
  setPageFeedback,
}: RuntimeControlHandlerOptions) {
  return (agent: AgentInstance) =>
    runAgentRuntimeControl({
      agent,
      projectId: getProjectId(),
      orgId: getOrgId(),
      action,
      loadData,
      successMessage: `${successPrefix} ${agent.name}.`,
      failureMessage,
      setPendingAgentId,
      setPageError,
      setPageFeedback,
    })
}
