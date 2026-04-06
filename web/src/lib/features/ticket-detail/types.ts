import type { TicketPickupDiagnosis } from './pickup-diagnosis'
export type {
  TicketRunTranscriptBlock,
  TicketRunTranscriptInterruptOption,
  TicketRunTranscriptState,
} from './run-transcript-types'

export type { TicketPickupDiagnosis } from './pickup-diagnosis'

export type TicketStatusOption = {
  id: string
  name: string
  color: string
  stage?: string
}

export type TicketReferenceOption = {
  id: string
  identifier: string
  title: string
}

export type TicketRepoOption = {
  id: string
  name: string
  defaultBranch: string
}

export type TicketExternalLink = {
  id: string
  type: string
  url: string
  externalId: string
  title?: string
  status?: string
  relation: string
}

export type TicketExternalLinkDraft = {
  type: string
  url: string
  externalId: string
  title: string
  status: string
  relation: string
}

export type TicketDetail = {
  id: string
  identifier: string
  title: string
  description: string
  archived: boolean
  status: TicketStatusOption
  priority: '' | 'urgent' | 'high' | 'medium' | 'low'
  type: 'feature' | 'bugfix' | 'refactor' | 'chore'
  workflow?: { id: string; name: string; type: string }
  assignedAgent?: {
    id: string
    name: string
    provider: string
    runtimeControlState?: string
    runtimePhase?: string
  }
  repoScopes: Array<{
    id: string
    repoId: string
    repoName: string
    branchName: string
    defaultBranch: string
    effectiveBranchName: string
    branchSource: 'generated' | 'override'
    prUrl?: string
  }>
  attemptCount: number
  consecutiveErrors: number
  retryPaused: boolean
  pauseReason?: string
  currentRunId?: string
  targetMachineId?: string
  nextRetryAt?: string
  costTokensInput: number
  costTokensOutput: number
  costTokensTotal: number
  costAmount: number
  budgetUsd: number
  pickupDiagnosis?: TicketPickupDiagnosis
  dependencies: Array<{
    id: string
    targetId: string
    identifier: string
    title: string
    relation: 'blocks' | 'blocked_by' | 'sub_issue'
    stage: 'backlog' | 'unstarted' | 'started' | 'completed' | 'canceled'
  }>
  externalLinks: TicketExternalLink[]
  children: Array<{ id: string; identifier: string; title: string; status: string }>
  createdBy: string
  createdAt: string
  updatedAt: string
  startedAt?: string
  completedAt?: string
}

export type TicketComment = {
  id: string
  ticketId: string
  body: string
  createdBy: string
  createdAt: string
  updatedAt?: string
}

export type TicketCommentRevision = {
  id: string
  commentId: string
  revisionNumber: number
  bodyMarkdown: string
  editedBy: string
  editedAt: string
  editReason?: string
}

export type TicketTimelineActor = {
  name: string
  type: string
}

export type TicketTimelineBase = {
  id: string
  ticketId: string
  actor: TicketTimelineActor
  createdAt: string
  updatedAt: string
  editedAt?: string
  isCollapsible: boolean
  isDeleted: boolean
}

export type TicketDescriptionTimelineItem = TicketTimelineBase & {
  kind: 'description'
  title: string
  bodyMarkdown: string
  identifier?: string
}

export type TicketCommentTimelineItem = TicketTimelineBase & {
  kind: 'comment'
  commentId: string
  bodyMarkdown: string
  editCount: number
  revisionCount: number
  lastEditedBy?: string
}

export type TicketActivityTimelineItem = TicketTimelineBase & {
  kind: 'activity'
  eventType: string
  title: string
  bodyText: string
  metadata: Record<string, unknown>
}

export type TicketTimelineItem =
  | TicketDescriptionTimelineItem
  | TicketCommentTimelineItem
  | TicketActivityTimelineItem

export type HookExecution = {
  id: string
  hookName: string
  status: 'pass' | 'fail' | 'running' | 'timeout'
  duration?: number
  output?: string
  timestamp: string
}

export type TicketActivity = {
  id: string
  type: string
  message: string
  timestamp: string
  agentName?: string
}

export type TicketRunUsage = {
  total: number
  input: number
  output: number
  cachedInput: number
  cacheCreation: number
  reasoning: number
  prompt: number
  candidate: number
  tool: number
}

export type TicketRun = {
  id: string
  attemptNumber: number
  agentId: string
  agentName: string
  provider: string
  adapterType: string
  modelName: string
  usage: TicketRunUsage
  status: 'launching' | 'ready' | 'executing' | 'ended' | 'failed' | 'interrupted' | 'completed'
  currentStepStatus?: string
  currentStepSummary?: string
  createdAt: string
  runtimeStartedAt?: string
  lastHeartbeatAt?: string
  terminalAt?: string
  completedAt?: string
  lastError?: string
  completionSummary?: TicketRunCompletionSummary
}

export type TicketRunCompletionSummary = {
  status: 'pending' | 'completed' | 'failed'
  markdown?: string
  json?: Record<string, unknown>
  generatedAt?: string
  error?: string
}

export type TicketRunTraceEntry = {
  id: string
  agentRunId: string
  sequence: number
  provider: string
  kind: string
  stream: string
  output: string
  payload: Record<string, unknown>
  createdAt: string
}

export type TicketRunStepEntry = {
  id: string
  agentRunId: string
  stepStatus: string
  summary: string
  sourceTraceEventId?: string
  createdAt: string
}

export type TicketRunTranscriptItem =
  | {
      kind: 'trace'
      cursor: string
      traceEntry: TicketRunTraceEntry
    }
  | {
      kind: 'step'
      cursor: string
      stepEntry: TicketRunStepEntry
    }

export type TicketRunTranscriptPage = {
  items: TicketRunTranscriptItem[]
  hasOlder: boolean
  hiddenOlderCount: number
  hasNewer: boolean
  hiddenNewerCount: number
  oldestCursor?: string
  newestCursor?: string
}

export type TicketRunDetail = {
  run: TicketRun
  transcriptPage: TicketRunTranscriptPage
}

export type TicketRunLifecycleEvent = {
  eventType: string
  message: string
  createdAt: string
}
