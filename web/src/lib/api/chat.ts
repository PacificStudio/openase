import { ApiError, buildRequestHeaders } from './client'
import { consumeEventStream, defaultActivityTimeoutMs, type SSEFrame } from './sse'
import type { ProjectAIFocus } from '$lib/features/chat/project-ai-focus'

export type ChatSource = 'project_sidebar' | 'ticket_detail'

export type ChatTurnRequest = {
  message: string
  source: ChatSource
  providerId?: string
  sessionId?: string
  context: {
    projectId: string
    ticketId?: string
  }
}

export type ChatTextPayload = {
  type: 'text'
  content: string
}

export type ChatDonePayload = {
  sessionId: string
  turnsUsed: number
  turnsRemaining?: number
  costUSD?: number
}

export type ChatSessionPayload = {
  sessionId: string
}

export type ChatErrorPayload = {
  message: string
}

export type ChatDiffLineOp = 'context' | 'add' | 'remove'

export type ChatDiffLine = {
  op: ChatDiffLineOp
  text: string
}

export type ChatDiffHunk = {
  oldStart: number
  oldLines: number
  newStart: number
  newLines: number
  lines: ChatDiffLine[]
}

export type ChatDiffPayload = {
  type: 'diff'
  entryId?: string
  file: string
  hunks: ChatDiffHunk[]
}

export type ChatBundleDiffFile = {
  file: string
  hunks: ChatDiffHunk[]
}

export type ChatBundleDiffPayload = {
  type: 'bundle_diff'
  entryId?: string
  files: ChatBundleDiffFile[]
}

export type ChatTaskPayload = {
  type: string
  raw?: unknown
}

export type ChatMessagePayload =
  | ChatTextPayload
  | ChatDiffPayload
  | ChatBundleDiffPayload
  | ChatTaskPayload

export type ChatStreamEvent =
  | { kind: 'session'; payload: ChatSessionPayload }
  | { kind: 'message'; payload: ChatMessagePayload }
  | { kind: 'done'; payload: ChatDonePayload }
  | { kind: 'error'; payload: ChatErrorPayload }

export type ProjectConversation = {
  id: string
  projectId: string
  userId: string
  source: 'project_sidebar'
  providerId: string
  title: string
  providerAnchorKind?: 'thread' | 'session'
  providerAnchorId?: string
  providerTurnId?: string
  providerTurnSupported?: boolean
  providerStatus?: string
  providerActiveFlags?: string[]
  status: string
  rollingSummary: string
  lastActivityAt: string
  createdAt: string
  updatedAt: string
  runtimePrincipal?: ProjectConversationRuntimePrincipal
}

export type ProjectConversationRuntimePrincipal = {
  id?: string
  name?: string
  status?: string
  runtimeState?: string
  currentSessionId?: string
  currentWorkspacePath?: string
  currentRunId?: string
  lastHeartbeatAt?: string
  currentStepStatus?: string
  currentStepSummary?: string
  currentStepChangedAt?: string
}

export type ProjectConversationEntry = {
  id: string
  conversationId: string
  turnId?: string
  seq: number
  kind: string
  payload: Record<string, unknown>
  createdAt: string
}

export type ProjectConversationWorkspaceFileStatus =
  | 'modified'
  | 'added'
  | 'deleted'
  | 'renamed'
  | 'untracked'

export type ProjectConversationWorkspaceDiffFile = {
  path: string
  oldPath?: string
  status: ProjectConversationWorkspaceFileStatus
  staged?: boolean
  unstaged?: boolean
  added: number
  removed: number
}

export type ProjectConversationWorkspaceMissingRepo = {
  name: string
  path: string
}

export type ProjectConversationWorkspaceSyncReason = 'repo_binding_changed' | 'repo_missing'

export type ProjectConversationWorkspaceSyncPrompt = {
  reason: ProjectConversationWorkspaceSyncReason
  missingRepos: ProjectConversationWorkspaceMissingRepo[]
}

export type ProjectConversationWorkspaceDiffRepo = {
  name: string
  path: string
  branch: string
  dirty: boolean
  filesChanged: number
  added: number
  removed: number
  files: ProjectConversationWorkspaceDiffFile[]
}

export type ProjectConversationWorkspaceDiff = {
  conversationId: string
  workspacePath: string
  dirty: boolean
  reposChanged: number
  filesChanged: number
  added: number
  removed: number
  repos: ProjectConversationWorkspaceDiffRepo[]
  syncPrompt?: ProjectConversationWorkspaceSyncPrompt
}

export type ProjectConversationWorkspaceRepoMetadata = {
  name: string
  path: string
  branch: string
  currentRef: ProjectConversationWorkspaceCurrentRef
  headCommit: string
  headSummary: string
  dirty: boolean
  filesChanged: number
  added: number
  removed: number
}

export type ProjectConversationWorkspaceCurrentRefKind = 'branch' | 'detached'

export type ProjectConversationWorkspaceCurrentRef = {
  kind: ProjectConversationWorkspaceCurrentRefKind
  displayName: string
  cacheKey: string
  branchName: string
  branchFullName: string
  commitId: string
  shortCommitId: string
  subject: string
}

export type ProjectConversationWorkspaceBranchScope = 'local_branch' | 'remote_tracking_branch'

export type ProjectConversationWorkspaceBranchRef = {
  name: string
  fullName: string
  scope: ProjectConversationWorkspaceBranchScope
  current: boolean
  commitId: string
  shortCommitId: string
  subject: string
  upstreamName: string
  ahead: number
  behind: number
  suggestedLocalBranchName: string
}

export type ProjectConversationWorkspaceRepoRefs = {
  conversationId: string
  repoPath: string
  currentRef: ProjectConversationWorkspaceCurrentRef
  localBranches: ProjectConversationWorkspaceBranchRef[]
  remoteBranches: ProjectConversationWorkspaceBranchRef[]
}

export type ProjectConversationWorkspaceGitRefLabelScope =
  | 'head'
  | 'local_branch'
  | 'remote_tracking_branch'

export type ProjectConversationWorkspaceGitRefLabel = {
  name: string
  fullName: string
  scope: ProjectConversationWorkspaceGitRefLabelScope
  current: boolean
}

export type ProjectConversationWorkspaceGitGraphCommit = {
  commitId: string
  shortCommitId: string
  parentIds: string[]
  subject: string
  authorName: string
  authoredAt: string
  labels: ProjectConversationWorkspaceGitRefLabel[]
  head: boolean
}

export type ProjectConversationWorkspaceGitGraph = {
  conversationId: string
  repoPath: string
  limit: number
  commits: ProjectConversationWorkspaceGitGraphCommit[]
}

export type ProjectConversationWorkspaceCheckoutResult = {
  conversationId: string
  repoPath: string
  currentRef: ProjectConversationWorkspaceCurrentRef
  createdLocalBranch: string
}

export type ProjectConversationWorkspaceMetadata = {
  conversationId: string
  available: boolean
  workspacePath: string
  repos: ProjectConversationWorkspaceRepoMetadata[]
  syncPrompt?: ProjectConversationWorkspaceSyncPrompt
}

export type ProjectConversationWorkspaceTreeEntryKind = 'directory' | 'file'

export type ProjectConversationWorkspaceTreeEntry = {
  path: string
  name: string
  kind: ProjectConversationWorkspaceTreeEntryKind
  sizeBytes: number
}

export type ProjectConversationWorkspaceTree = {
  conversationId: string
  repoPath: string
  path: string
  entries: ProjectConversationWorkspaceTreeEntry[]
}

export type ProjectConversationWorkspaceSearchResult = {
  path: string
  name: string
}

export type ProjectConversationWorkspaceSearch = {
  conversationId: string
  repoPath: string
  query: string
  truncated: boolean
  results: ProjectConversationWorkspaceSearchResult[]
}

export type ProjectConversationWorkspacePreviewKind = 'text' | 'binary'

export type ProjectConversationWorkspaceFilePreview = {
  conversationId: string
  repoPath: string
  path: string
  sizeBytes: number
  mediaType: string
  previewKind: ProjectConversationWorkspacePreviewKind
  truncated: boolean
  content: string
  revision: string
  writable: boolean
  readOnlyReason: string
  encoding: 'utf-8'
  lineEnding: 'lf' | 'crlf'
}

export type ProjectConversationWorkspaceDiffKind = 'none' | 'text' | 'binary'

export type ProjectConversationWorkspaceFilePatch = {
  conversationId: string
  repoPath: string
  path: string
  status: ProjectConversationWorkspaceFileStatus
  diffKind: ProjectConversationWorkspaceDiffKind
  truncated: boolean
  diff: string
}

export type ProjectConversationTerminalMode = 'shell'

export type ProjectConversationTerminalSession = {
  id: string
  mode: ProjectConversationTerminalMode
  cwd: string
  wsPath: string
  attachToken: string
}

export type ProjectConversationTerminalSessionRequest = {
  mode: ProjectConversationTerminalMode
  repoPath?: string
  cwdPath?: string
  cols?: number
  rows?: number
}

export type ProjectConversationInterruptOption = {
  id: string
  label: string
}

export type ProjectConversationInterruptRequestedPayload = {
  interruptId: string
  provider: string
  kind: string
  options: ProjectConversationInterruptOption[]
  payload: Record<string, unknown>
}

export type ProjectConversationInterruptResolvedPayload = {
  interruptId: string
  decision?: string
}

export type ProjectConversationTurnRequest = {
  message: string
  focus?: ProjectAIFocus | null
}

export type ProjectConversationWorkspaceFileSaveRequest = {
  repoPath: string
  path: string
  baseRevision: string
  content: string
  encoding: 'utf-8'
  lineEnding: 'lf' | 'crlf'
}

export type ProjectConversationWorkspaceFileSaved = {
  conversationId: string
  repoPath: string
  path: string
  revision: string
  sizeBytes: number
  encoding: 'utf-8'
  lineEnding: 'lf' | 'crlf'
}

export type ProjectConversationWorkspaceFileCreateRequest = {
  repoPath: string
  path: string
}

export type ProjectConversationWorkspaceFileCreated = {
  conversationId: string
  repoPath: string
  path: string
  revision: string
  sizeBytes: number
  encoding: 'utf-8'
  lineEnding: 'lf' | 'crlf'
}

export type ProjectConversationWorkspaceFileRenameRequest = {
  repoPath: string
  fromPath: string
  toPath: string
}

export type ProjectConversationWorkspaceFileRenamed = {
  conversationId: string
  repoPath: string
  fromPath: string
  toPath: string
}

export type ProjectConversationWorkspaceFileDeleteRequest = {
  repoPath: string
  path: string
}

export type ProjectConversationWorkspaceFileDeleted = {
  conversationId: string
  repoPath: string
  path: string
}

export type ProjectConversationTurnDonePayload = {
  conversationId: string
  turnId: string
  costUSD?: number
}

export type ProjectConversationInterruptedPayload = {
  conversationId?: string
  turnId?: string
  message: string
  reason?: string
}

export type ProjectConversationTurnResponse = {
  turn: {
    id: string
    turnIndex: number
    status: string
  }
  conversation: ProjectConversation
}

export type ProjectConversationReasoningUpdatedPayload = {
  threadId: string
  turnId: string
  itemId: string
  kind: string
  delta?: string
  summaryIndex?: number
  contentIndex?: number
  entryId?: string
}

export type ProjectConversationDiffUpdatedPayload = {
  threadId: string
  turnId: string
  diff: string
  entryId?: string
}

export type ProjectConversationSessionPayload = {
  conversationId: string
  runtimeState: string
  title?: string
  rollingSummary?: string
  providerAnchorKind?: 'thread' | 'session'
  providerAnchorId?: string
  providerTurnId?: string
  providerTurnSupported?: boolean
  providerStatus?: string
  providerActiveFlags: string[]
}

export type ProjectConversationStreamEvent =
  | { kind: 'session'; payload: ProjectConversationSessionPayload }
  | { kind: 'message'; payload: ChatMessagePayload }
  | { kind: 'interrupt_requested'; payload: ProjectConversationInterruptRequestedPayload }
  | { kind: 'interrupt_resolved'; payload: ProjectConversationInterruptResolvedPayload }
  | { kind: 'diff_updated'; payload: ProjectConversationDiffUpdatedPayload }
  | { kind: 'reasoning_updated'; payload: ProjectConversationReasoningUpdatedPayload }
  | { kind: 'interrupted'; payload: ProjectConversationInterruptedPayload }
  | { kind: 'turn_done'; payload: ProjectConversationTurnDonePayload }
  | { kind: 'error'; payload: ChatErrorPayload }

export type ProjectConversationMuxFrame = {
  conversationId: string
  sentAt: string
  event: ProjectConversationStreamEvent
}

export type ProjectConversationMuxFrameParseResult =
  | { ok: true; value: ProjectConversationMuxFrame }
  | { ok: false; error: Error }

type RawProjectConversation = {
  id?: string
  project_id?: string
  user_id?: string
  source?: string
  provider_id?: string
  status?: string
  title?: string
  provider_anchor_kind?: string
  provider_anchor_id?: string
  provider_turn_id?: string
  provider_turn_supported?: boolean
  provider_status?: string
  provider_active_flags?: string[]
  rolling_summary?: string
  last_activity_at?: string
  created_at?: string
  updated_at?: string
  runtime_principal?: Record<string, unknown>
}

type RawProjectConversationEntry = {
  id?: string
  conversation_id?: string
  turn_id?: string
  seq?: number
  kind?: string
  payload?: Record<string, unknown>
  created_at?: string
}

export async function streamChatTurn(
  request: ChatTurnRequest,
  handlers: {
    signal?: AbortSignal
    onEvent: (event: ChatStreamEvent) => void
  },
) {
  const headers = buildRequestHeaders('POST', {
    accept: 'text/event-stream',
    'Content-Type': 'application/json',
  })
  const response = await fetch('/api/v1/chat', {
    method: 'POST',
    headers,
    body: JSON.stringify({
      message: request.message,
      source: request.source,
      provider_id: request.providerId,
      session_id: request.sessionId,
      context: {
        project_id: request.context.projectId,
        ticket_id: request.context.ticketId,
      },
    }),
    credentials: 'same-origin',
    signal: handlers.signal,
  })

  if (!response.ok) {
    const detail = await response.text().catch(() => response.statusText)
    throw new ApiError(response.status, detail)
  }
  if (!response.body) {
    throw new Error('chat stream response body is unavailable')
  }

  await consumeEventStream(response.body, (frame) => {
    const event = parseChatStreamEvent(frame)
    if (event) {
      handlers.onEvent(event)
    }
  })
}

export async function closeChatSession(sessionId: string) {
  const headers = buildRequestHeaders('DELETE')
  const response = await fetch(`/api/v1/chat/${encodeURIComponent(sessionId)}`, {
    method: 'DELETE',
    headers,
    credentials: 'same-origin',
  })

  if (!response.ok) {
    const detail = await response.text().catch(() => response.statusText)
    throw new ApiError(response.status, detail)
  }
}

export function createProjectConversation(request: { providerId: string; projectId: string }) {
  return fetchJSON<{ conversation?: RawProjectConversation }>('/api/v1/chat/conversations', {
    method: 'POST',
    body: {
      source: 'project_sidebar',
      provider_id: request.providerId,
      context: { project_id: request.projectId },
    },
  }).then((payload) => ({
    conversation: parseProjectConversation(payload.conversation),
  }))
}

export function listProjectConversations(request: { projectId: string; providerId?: string }) {
  return fetchJSON<{ conversations?: RawProjectConversation[] }>('/api/v1/chat/conversations', {
    params: {
      project_id: request.projectId,
      provider_id: request.providerId,
    },
  }).then((payload) => ({
    conversations: Array.isArray(payload.conversations)
      ? payload.conversations.map((conversation) => parseProjectConversation(conversation))
      : [],
  }))
}

export function getProjectConversation(conversationId: string) {
  return fetchJSON<{ conversation?: RawProjectConversation }>(
    `/api/v1/chat/conversations/${encodeURIComponent(conversationId)}`,
  ).then((payload) => ({
    conversation: parseProjectConversation(payload.conversation),
  }))
}

export function listProjectConversationEntries(conversationId: string) {
  return fetchJSON<{ entries?: RawProjectConversationEntry[] }>(
    `/api/v1/chat/conversations/${encodeURIComponent(conversationId)}/entries`,
  ).then((payload) => ({
    entries: Array.isArray(payload.entries)
      ? payload.entries.map((entry) => parseProjectConversationEntry(entry))
      : [],
  }))
}

export async function getProjectConversationWorkspaceDiff(conversationId: string) {
  const payload = await fetchJSON<{ workspace_diff?: unknown }>(
    `/api/v1/chat/conversations/${encodeURIComponent(conversationId)}/workspace-diff`,
  )
  const object = parseRequiredObject(payload as Record<string, unknown>)
  return {
    workspaceDiff: parseProjectConversationWorkspaceDiff(
      object.workspace_diff ?? object.workspaceDiff ?? object,
    ),
  }
}

export async function getProjectConversationWorkspace(conversationId: string) {
  const payload = await fetchJSON<{ workspace?: unknown }>(
    `/api/v1/chat/conversations/${encodeURIComponent(conversationId)}/workspace`,
  )
  const object = parseRequiredObject(payload as Record<string, unknown>)
  return {
    workspace: parseProjectConversationWorkspaceMetadata(object.workspace ?? object),
  }
}

export async function syncProjectConversationWorkspace(conversationId: string) {
  const payload = await fetchJSON<{ workspace?: unknown }>(
    `/api/v1/chat/conversations/${encodeURIComponent(conversationId)}/workspace/sync`,
    { method: 'POST' },
  )
  const object = parseRequiredObject(payload as Record<string, unknown>)
  return {
    workspace: parseProjectConversationWorkspaceMetadata(object.workspace ?? object),
  }
}

export async function getProjectConversationWorkspaceRepoRefs(
  conversationId: string,
  request: { repoPath: string },
) {
  const payload = await fetchJSON<{ repo_refs?: unknown }>(
    `/api/v1/chat/conversations/${encodeURIComponent(conversationId)}/workspace/repo-refs`,
    {
      params: {
        repo_path: request.repoPath,
      },
    },
  )
  const object = parseRequiredObject(payload as Record<string, unknown>)
  return {
    repoRefs: parseProjectConversationWorkspaceRepoRefs(object.repo_refs ?? object),
  }
}

export async function getProjectConversationWorkspaceGitGraph(
  conversationId: string,
  request: { repoPath: string; limit?: number },
) {
  const payload = await fetchJSON<{ git_graph?: unknown }>(
    `/api/v1/chat/conversations/${encodeURIComponent(conversationId)}/workspace/git-graph`,
    {
      params: {
        repo_path: request.repoPath,
        limit: request.limit == null ? undefined : String(request.limit),
      },
    },
  )
  const object = parseRequiredObject(payload as Record<string, unknown>)
  return {
    gitGraph: parseProjectConversationWorkspaceGitGraph(object.git_graph ?? object),
  }
}

export async function checkoutProjectConversationWorkspaceBranch(
  conversationId: string,
  request: {
    repoPath: string
    targetKind: ProjectConversationWorkspaceBranchScope
    targetName: string
    createTrackingBranch: boolean
    localBranchName?: string
    expectedCleanWorkspace: boolean
  },
) {
  const payload = await fetchJSON<{ checkout?: unknown }>(
    `/api/v1/chat/conversations/${encodeURIComponent(conversationId)}/workspace/checkout`,
    {
      method: 'POST',
      body: {
        repo_path: request.repoPath,
        target_kind: request.targetKind,
        target_name: request.targetName,
        create_tracking_branch: request.createTrackingBranch,
        local_branch_name: request.localBranchName ?? '',
        expected_clean_workspace: request.expectedCleanWorkspace,
      },
    },
  )
  const object = parseRequiredObject(payload as Record<string, unknown>)
  return {
    checkout: parseProjectConversationWorkspaceCheckoutResult(object.checkout ?? object),
  }
}

export type ProjectConversationWorkspaceGitRemoteOp = 'fetch' | 'pull' | 'push'

export type ProjectConversationWorkspaceGitRemoteOpResult = {
  conversationId: string
  repoPath: string
  op: ProjectConversationWorkspaceGitRemoteOp
  output: string
}

export type ProjectConversationWorkspaceGitStageResult = {
  conversationId: string
  repoPath: string
  path: string
}

export type ProjectConversationWorkspaceGitStageAllResult = {
  conversationId: string
  repoPath: string
}

export type ProjectConversationWorkspaceGitCommitResult = {
  conversationId: string
  repoPath: string
  output: string
}

export type ProjectConversationWorkspaceGitDiscardResult = {
  conversationId: string
  repoPath: string
  path: string
}

export type ProjectConversationWorkspaceGitUnstageResult = {
  conversationId: string
  repoPath: string
  path: string
}

export async function runProjectConversationWorkspaceGitRemoteOp(
  conversationId: string,
  request: { repoPath: string; op: ProjectConversationWorkspaceGitRemoteOp },
): Promise<ProjectConversationWorkspaceGitRemoteOpResult> {
  const payload = await fetchJSON<Record<string, unknown>>(
    `/api/v1/chat/conversations/${encodeURIComponent(conversationId)}/workspace/git-remote-op`,
    {
      method: 'POST',
      body: {
        repo_path: request.repoPath,
        op: request.op,
      },
    },
  )
  return {
    conversationId: readRequiredString(payload, 'conversation_id'),
    repoPath: readRequiredString(payload, 'repo_path'),
    op: readRequiredString(payload, 'op') as ProjectConversationWorkspaceGitRemoteOp,
    output: readOptionalString(payload, 'output') ?? '',
  }
}

export async function stageProjectConversationWorkspaceFile(
  conversationId: string,
  request: { repoPath: string; path: string },
): Promise<ProjectConversationWorkspaceGitStageResult> {
  const payload = await fetchJSON<Record<string, unknown>>(
    `/api/v1/chat/conversations/${encodeURIComponent(conversationId)}/workspace/git-stage`,
    {
      method: 'POST',
      body: {
        repo_path: request.repoPath,
        path: request.path,
      },
    },
  )
  return {
    conversationId: readRequiredString(payload, 'conversation_id'),
    repoPath: readRequiredString(payload, 'repo_path'),
    path: readRequiredString(payload, 'path'),
  }
}

export async function stageAllProjectConversationWorkspaceFiles(
  conversationId: string,
  request: { repoPath: string },
): Promise<ProjectConversationWorkspaceGitStageAllResult> {
  const payload = await fetchJSON<Record<string, unknown>>(
    `/api/v1/chat/conversations/${encodeURIComponent(conversationId)}/workspace/git-stage-all`,
    {
      method: 'POST',
      body: {
        repo_path: request.repoPath,
      },
    },
  )
  return {
    conversationId: readRequiredString(payload, 'conversation_id'),
    repoPath: readRequiredString(payload, 'repo_path'),
  }
}

export async function commitProjectConversationWorkspace(
  conversationId: string,
  request: { repoPath: string; message: string },
): Promise<ProjectConversationWorkspaceGitCommitResult> {
  const payload = await fetchJSON<Record<string, unknown>>(
    `/api/v1/chat/conversations/${encodeURIComponent(conversationId)}/workspace/git-commit`,
    {
      method: 'POST',
      body: {
        repo_path: request.repoPath,
        message: request.message,
      },
    },
  )
  return {
    conversationId: readRequiredString(payload, 'conversation_id'),
    repoPath: readRequiredString(payload, 'repo_path'),
    output: readOptionalString(payload, 'output') ?? '',
  }
}

export async function discardProjectConversationWorkspaceFile(
  conversationId: string,
  request: { repoPath: string; path: string },
): Promise<ProjectConversationWorkspaceGitDiscardResult> {
  const payload = await fetchJSON<Record<string, unknown>>(
    `/api/v1/chat/conversations/${encodeURIComponent(conversationId)}/workspace/git-discard`,
    {
      method: 'POST',
      body: {
        repo_path: request.repoPath,
        path: request.path,
      },
    },
  )
  return {
    conversationId: readRequiredString(payload, 'conversation_id'),
    repoPath: readRequiredString(payload, 'repo_path'),
    path: readRequiredString(payload, 'path'),
  }
}

export async function unstageProjectConversationWorkspace(
  conversationId: string,
  request: { repoPath: string; path?: string },
): Promise<ProjectConversationWorkspaceGitUnstageResult> {
  const payload = await fetchJSON<Record<string, unknown>>(
    `/api/v1/chat/conversations/${encodeURIComponent(conversationId)}/workspace/git-unstage`,
    {
      method: 'POST',
      body: {
        repo_path: request.repoPath,
        path: request.path ?? '',
      },
    },
  )
  return {
    conversationId: readRequiredString(payload, 'conversation_id'),
    repoPath: readRequiredString(payload, 'repo_path'),
    path: readOptionalString(payload, 'path') ?? '',
  }
}

export type ProjectConversationWorkspaceCreateBranchResult = {
  conversationId: string
  repoPath: string
  branchName: string
}

export async function createProjectConversationWorkspaceBranch(
  conversationId: string,
  request: { repoPath: string; branchName: string; startPoint?: string },
): Promise<ProjectConversationWorkspaceCreateBranchResult> {
  const payload = await fetchJSON<Record<string, unknown>>(
    `/api/v1/chat/conversations/${encodeURIComponent(conversationId)}/workspace/create-branch`,
    {
      method: 'POST',
      body: {
        repo_path: request.repoPath,
        branch_name: request.branchName,
        start_point: request.startPoint ?? '',
      },
    },
  )
  return {
    conversationId: readRequiredString(payload, 'conversation_id'),
    repoPath: readRequiredString(payload, 'repo_path'),
    branchName: readRequiredString(payload, 'branch_name'),
  }
}

export async function listProjectConversationWorkspaceTree(
  conversationId: string,
  request: { repoPath: string; path?: string },
) {
  const payload = await fetchJSON<{ workspace_tree?: unknown }>(
    `/api/v1/chat/conversations/${encodeURIComponent(conversationId)}/workspace/tree`,
    {
      params: {
        repo_path: request.repoPath,
        path: request.path,
      },
    },
  )
  const object = parseRequiredObject(payload as Record<string, unknown>)
  return {
    workspaceTree: parseProjectConversationWorkspaceTree(object.workspace_tree ?? object),
  }
}

export async function searchProjectConversationWorkspacePaths(
  conversationId: string,
  request: { repoPath: string; query: string; limit?: number },
) {
  const payload = await fetchJSON<{ workspace_search?: unknown }>(
    `/api/v1/chat/conversations/${encodeURIComponent(conversationId)}/workspace/search`,
    {
      params: {
        repo_path: request.repoPath,
        q: request.query,
        limit: request.limit == null ? undefined : String(request.limit),
      },
    },
  )
  const object = parseRequiredObject(payload as Record<string, unknown>)
  return {
    workspaceSearch: parseProjectConversationWorkspaceSearch(object.workspace_search ?? object),
  }
}

export async function getProjectConversationWorkspaceFilePreview(
  conversationId: string,
  request: { repoPath: string; path: string },
) {
  const payload = await fetchJSON<{ file_preview?: unknown }>(
    `/api/v1/chat/conversations/${encodeURIComponent(conversationId)}/workspace/file`,
    {
      params: {
        repo_path: request.repoPath,
        path: request.path,
      },
    },
  )
  const object = parseRequiredObject(payload as Record<string, unknown>)
  return {
    filePreview: parseProjectConversationWorkspaceFilePreview(object.file_preview ?? object),
  }
}

export async function saveProjectConversationWorkspaceFile(
  conversationId: string,
  request: ProjectConversationWorkspaceFileSaveRequest,
) {
  const payload = await fetchJSON<{ file?: unknown }>(
    `/api/v1/chat/conversations/${encodeURIComponent(conversationId)}/workspace/file`,
    {
      method: 'PUT',
      body: {
        repo_path: request.repoPath,
        path: request.path,
        base_revision: request.baseRevision,
        content: request.content,
        encoding: request.encoding,
        line_ending: request.lineEnding,
      },
    },
  )
  const object = parseRequiredObject(payload as Record<string, unknown>)
  return {
    file: parseProjectConversationWorkspaceFileSaved(object.file ?? object),
  }
}

export async function createProjectConversationWorkspaceFile(
  conversationId: string,
  request: ProjectConversationWorkspaceFileCreateRequest,
) {
  const payload = await fetchJSON<{ file?: unknown }>(
    `/api/v1/chat/conversations/${encodeURIComponent(conversationId)}/workspace/file`,
    {
      method: 'POST',
      body: {
        repo_path: request.repoPath,
        path: request.path,
      },
    },
  )
  const object = parseRequiredObject(payload as Record<string, unknown>)
  return {
    file: parseProjectConversationWorkspaceFileCreated(object.file ?? object),
  }
}

export async function renameProjectConversationWorkspaceFile(
  conversationId: string,
  request: ProjectConversationWorkspaceFileRenameRequest,
) {
  const payload = await fetchJSON<{ file?: unknown }>(
    `/api/v1/chat/conversations/${encodeURIComponent(conversationId)}/workspace/file`,
    {
      method: 'PATCH',
      body: {
        repo_path: request.repoPath,
        from_path: request.fromPath,
        to_path: request.toPath,
      },
    },
  )
  const object = parseRequiredObject(payload as Record<string, unknown>)
  return {
    file: parseProjectConversationWorkspaceFileRenamed(object.file ?? object),
  }
}

export async function deleteProjectConversationWorkspaceFile(
  conversationId: string,
  request: ProjectConversationWorkspaceFileDeleteRequest,
) {
  const payload = await fetchJSON<{ file?: unknown }>(
    `/api/v1/chat/conversations/${encodeURIComponent(conversationId)}/workspace/file`,
    {
      method: 'DELETE',
      body: {
        repo_path: request.repoPath,
        path: request.path,
      },
    },
  )
  const object = parseRequiredObject(payload as Record<string, unknown>)
  return {
    file: parseProjectConversationWorkspaceFileDeleted(object.file ?? object),
  }
}

export async function getProjectConversationWorkspaceFilePatch(
  conversationId: string,
  request: { repoPath: string; path: string },
) {
  const payload = await fetchJSON<{ file_patch?: unknown }>(
    `/api/v1/chat/conversations/${encodeURIComponent(conversationId)}/workspace/file-patch`,
    {
      params: {
        repo_path: request.repoPath,
        path: request.path,
      },
    },
  )
  const object = parseRequiredObject(payload as Record<string, unknown>)
  return {
    filePatch: parseProjectConversationWorkspaceFilePatch(object.file_patch ?? object),
  }
}

export async function createProjectConversationTerminalSession(
  conversationId: string,
  request: ProjectConversationTerminalSessionRequest,
) {
  const payload = await fetchJSON<{ terminal_session?: unknown }>(
    `/api/v1/chat/conversations/${encodeURIComponent(conversationId)}/terminal-sessions`,
    {
      method: 'POST',
      body: {
        mode: request.mode,
        repo_path: request.repoPath,
        cwd_path: request.cwdPath,
        cols: request.cols,
        rows: request.rows,
      },
    },
  )
  const object = parseRequiredObject(payload as Record<string, unknown>)
  return {
    terminalSession: parseProjectConversationTerminalSession(object.terminal_session ?? object),
  }
}

export function startProjectConversationTurn(
  conversationId: string,
  request: ProjectConversationTurnRequest,
) {
  const workspaceFileDraft = serializeProjectConversationWorkspaceFileDraft(request.focus)
  return fetchJSON<{ turn?: Record<string, unknown>; conversation?: RawProjectConversation }>(
    `/api/v1/chat/conversations/${encodeURIComponent(conversationId)}/turns`,
    {
      method: 'POST',
      body: {
        message: request.message,
        focus: serializeProjectConversationFocus(request.focus),
        workspace_file_draft: workspaceFileDraft,
      },
    },
  ).then((payload) => {
    const object = parseRequiredObject(payload)
    const turn = parseRequiredObject(object.turn)
    return {
      turn: {
        id: readRequiredString(turn, 'id'),
        turnIndex: readRequiredNumber(turn, 'turn_index'),
        status: readRequiredString(turn, 'status'),
      },
      conversation: parseProjectConversation(object.conversation),
    } satisfies ProjectConversationTurnResponse
  })
}

function serializeProjectConversationFocus(focus: ProjectAIFocus | null | undefined) {
  if (!focus) {
    return undefined
  }

  switch (focus.kind) {
    case 'workflow':
      return {
        kind: focus.kind,
        workflow_id: focus.workflowId,
        workflow_name: focus.workflowName,
        workflow_type: focus.workflowType,
        harness_path: focus.harnessPath,
        is_active: focus.isActive,
        selected_area: focus.selectedArea,
        has_dirty_draft: focus.hasDirtyDraft,
      }
    case 'skill':
      return {
        kind: focus.kind,
        skill_id: focus.skillId,
        skill_name: focus.skillName,
        selected_file_path: focus.selectedFilePath,
        bound_workflow_names: focus.boundWorkflowNames,
        has_dirty_draft: focus.hasDirtyDraft,
      }
    case 'workspace_file':
      return {
        kind: focus.kind,
        conversation_id: focus.conversationId,
        workspace_repo_path: focus.repoPath,
        workspace_file_path: focus.filePath,
        selected_area: focus.selectedArea,
        has_dirty_draft: focus.hasDirtyDraft,
        workspace_selection_from: focus.selection?.from,
        workspace_selection_to: focus.selection?.to,
        workspace_selection_start_line: focus.selection?.startLine,
        workspace_selection_start_column: focus.selection?.startColumn,
        workspace_selection_end_line: focus.selection?.endLine,
        workspace_selection_end_column: focus.selection?.endColumn,
        workspace_selection_text: focus.selection?.text,
        workspace_selection_context_before: focus.selection?.contextBefore,
        workspace_selection_context_after: focus.selection?.contextAfter,
        workspace_selection_truncated: focus.selection?.truncated,
        workspace_working_set: focus.workingSet?.map((item) => ({
          file_path: item.filePath,
          content_excerpt: item.contentExcerpt,
          dirty: item.dirty,
          truncated: item.truncated,
        })),
      }
    case 'ticket':
      return {
        kind: focus.kind,
        ticket_id: focus.ticketId,
        ticket_identifier: focus.ticketIdentifier,
        ticket_title: focus.ticketTitle,
        ticket_description: focus.ticketDescription,
        ticket_status: focus.ticketStatus,
        ticket_priority: focus.ticketPriority,
        ticket_attempt_count: focus.ticketAttemptCount,
        ticket_retry_paused: focus.ticketRetryPaused,
        ticket_pause_reason: focus.ticketPauseReason,
        ticket_dependencies: focus.ticketDependencies?.map((dependency) => ({
          identifier: dependency.identifier,
          title: dependency.title,
          relation: dependency.relation,
          status: dependency.status,
        })),
        ticket_repo_scopes: focus.ticketRepoScopes?.map((scope) => ({
          repo_id: scope.repoId,
          repo_name: scope.repoName,
          branch_name: scope.branchName,
          pull_request_url: scope.pullRequestUrl,
        })),
        ticket_recent_activity: focus.ticketRecentActivity?.map((activity) => ({
          event_type: activity.eventType,
          message: activity.message,
          created_at: activity.createdAt,
        })),
        ticket_hook_history: focus.ticketHookHistory?.map((hook) => ({
          hook_name: hook.hookName,
          status: hook.status,
          output: hook.output,
          timestamp: hook.timestamp,
        })),
        ticket_assigned_agent: focus.ticketAssignedAgent
          ? {
              id: focus.ticketAssignedAgent.id,
              name: focus.ticketAssignedAgent.name,
              provider: focus.ticketAssignedAgent.provider,
              runtime_control_state: focus.ticketAssignedAgent.runtimeControlState,
              runtime_phase: focus.ticketAssignedAgent.runtimePhase,
            }
          : undefined,
        ticket_current_run: focus.ticketCurrentRun
          ? {
              id: focus.ticketCurrentRun.id,
              attempt_number: focus.ticketCurrentRun.attemptNumber,
              status: focus.ticketCurrentRun.status,
              current_step_status: focus.ticketCurrentRun.currentStepStatus,
              current_step_summary: focus.ticketCurrentRun.currentStepSummary,
              last_error: focus.ticketCurrentRun.lastError,
            }
          : undefined,
        ticket_target_machine: focus.ticketTargetMachine
          ? {
              id: focus.ticketTargetMachine.id,
              name: focus.ticketTargetMachine.name,
              host: focus.ticketTargetMachine.host,
            }
          : undefined,
        selected_area: focus.selectedArea,
      }
    case 'machine':
      return {
        kind: focus.kind,
        machine_id: focus.machineId,
        machine_name: focus.machineName,
        machine_host: focus.machineHost,
        machine_status: focus.machineStatus,
        selected_area: focus.selectedArea,
        health_summary: focus.healthSummary,
      }
  }
}

function serializeProjectConversationWorkspaceFileDraft(focus: ProjectAIFocus | null | undefined) {
  if (
    focus?.kind !== 'workspace_file' ||
    !focus.hasDirtyDraft ||
    typeof focus.draftContent !== 'string' ||
    focus.draftContent.length === 0
  ) {
    return undefined
  }
  return {
    repo_path: focus.repoPath,
    path: focus.filePath,
    content: focus.draftContent,
    encoding: focus.encoding ?? 'utf-8',
    line_ending: focus.lineEnding ?? 'lf',
  }
}

export async function watchProjectConversation(
  conversationId: string,
  handlers: {
    signal?: AbortSignal
    onEvent: (event: ProjectConversationStreamEvent) => void
  },
) {
  const headers = buildRequestHeaders('GET', {
    accept: 'text/event-stream',
  })
  const response = await fetch(
    `/api/v1/chat/conversations/${encodeURIComponent(conversationId)}/stream`,
    {
      method: 'GET',
      headers,
      credentials: 'same-origin',
      signal: handlers.signal,
    },
  )

  if (!response.ok) {
    const detail = await response.text().catch(() => response.statusText)
    throw new ApiError(response.status, detail)
  }
  if (!response.body) {
    throw new Error('project conversation stream response body is unavailable')
  }

  await consumeEventStream(response.body, (frame) => {
    const event = parseProjectConversationStreamEvent(frame)
    if (event) {
      handlers.onEvent(event)
    }
  })
}

export async function watchProjectConversationMuxStream(
  projectId: string,
  handlers: {
    signal?: AbortSignal
    onOpen?: () => void
    onFrame: (frame: ProjectConversationMuxFrame) => void
    activityTimeoutMs?: number
  },
) {
  const headers = buildRequestHeaders('GET', {
    accept: 'text/event-stream',
  })
  const response = await fetch(
    `/api/v1/chat/projects/${encodeURIComponent(projectId)}/conversations/stream`,
    {
      method: 'GET',
      headers,
      credentials: 'same-origin',
      signal: handlers.signal,
    },
  )

  if (!response.ok) {
    const detail = await response.text().catch(() => response.statusText)
    throw new ApiError(response.status, detail)
  }
  if (!response.body) {
    throw new Error('project conversation mux stream response body is unavailable')
  }

  handlers.onOpen?.()
  await consumeEventStream(
    response.body,
    (frame) => {
      const parsed = parseRawProjectConversationMuxFrame(frame)
      if (parsed.ok) {
        handlers.onFrame(parsed.value)
      }
    },
    { activityTimeoutMs: handlers.activityTimeoutMs ?? defaultActivityTimeoutMs },
  )
}

export function respondProjectConversationInterrupt(
  conversationId: string,
  interruptId: string,
  body: {
    decision?: string | null
    answer?: Record<string, unknown> | null
  },
) {
  return fetchJSON<{ interrupt: Record<string, unknown> }>(
    `/api/v1/chat/conversations/${encodeURIComponent(conversationId)}/interrupts/${encodeURIComponent(interruptId)}/respond`,
    {
      method: 'POST',
      body: {
        decision: body.decision ?? undefined,
        answer: body.answer ?? undefined,
      },
    },
  )
}

export async function interruptProjectConversationTurn(conversationId: string) {
  const headers = buildRequestHeaders('POST')
  const response = await fetch(
    `/api/v1/chat/conversations/${encodeURIComponent(conversationId)}/interrupt-turn`,
    {
      method: 'POST',
      headers,
      credentials: 'same-origin',
    },
  )

  if (!response.ok) {
    const detail = await response.text().catch(() => response.statusText)
    throw new ApiError(response.status, detail)
  }
}

export async function closeProjectConversationRuntime(conversationId: string) {
  const headers = buildRequestHeaders('DELETE')
  const response = await fetch(
    `/api/v1/chat/conversations/${encodeURIComponent(conversationId)}/runtime`,
    {
      method: 'DELETE',
      headers,
      credentials: 'same-origin',
    },
  )

  if (!response.ok) {
    const detail = await response.text().catch(() => response.statusText)
    throw new ApiError(response.status, detail)
  }
}

export async function deleteProjectConversation(
  conversationId: string,
  request: { force?: boolean } = {},
) {
  const headers = buildRequestHeaders('DELETE')
  const params = new URLSearchParams()
  if (request.force) {
    params.set('force', 'true')
  }
  const query = params.size > 0 ? `?${params.toString()}` : ''
  const response = await fetch(
    `/api/v1/chat/conversations/${encodeURIComponent(conversationId)}${query}`,
    {
      method: 'DELETE',
      headers,
      credentials: 'same-origin',
    },
  )

  if (!response.ok) {
    const detail = await response.text().catch(() => response.statusText)
    throw new ApiError(response.status, detail)
  }
}

function parseChatStreamEvent(frame: SSEFrame): ChatStreamEvent | null {
  const payload = parseJSONObject(frame.data)
  if (payload == null) {
    return null
  }

  switch (frame.event) {
    case 'session':
      return { kind: 'session', payload: parseSessionPayload(payload) }
    case 'message':
      return { kind: 'message', payload: parseMessagePayload(payload) }
    case 'done':
      return { kind: 'done', payload: parseDonePayload(payload) }
    case 'error':
      return { kind: 'error', payload: parseErrorPayload(payload) }
    default:
      return null
  }
}

function parseProjectConversationStreamEvent(
  frame: SSEFrame,
): ProjectConversationStreamEvent | null {
  const payload = parseJSONObject(frame.data)
  if (payload == null) {
    return null
  }

  return parseProjectConversationStreamPayload(frame.event, payload)
}

function parseProjectConversationStreamPayload(
  eventName: string,
  payload: unknown,
): ProjectConversationStreamEvent | null {
  switch (eventName) {
    case 'session': {
      const object = parseRequiredObject(payload)
      return {
        kind: 'session',
        payload: {
          conversationId: readRequiredString(object, 'conversation_id'),
          runtimeState: readRequiredString(object, 'runtime_state'),
          title: readOptionalString(object, 'title'),
          rollingSummary: readOptionalString(object, 'rolling_summary'),
          providerAnchorKind: readProviderAnchorKind(object),
          providerAnchorId:
            readOptionalString(object, 'provider_anchor_id') ??
            readOptionalString(object, 'provider_thread_id'),
          providerTurnId:
            readOptionalString(object, 'provider_turn_id') ??
            readOptionalString(object, 'last_turn_id'),
          providerTurnSupported: readOptionalBoolean(object, 'provider_turn_supported'),
          providerStatus:
            readOptionalString(object, 'provider_status') ??
            readOptionalString(object, 'provider_thread_status'),
          providerActiveFlags:
            readOptionalStringList(object, 'provider_active_flags').length > 0
              ? readOptionalStringList(object, 'provider_active_flags')
              : readOptionalStringList(object, 'provider_thread_active_flags'),
        },
      }
    }
    case 'message':
      return { kind: 'message', payload: parseMessagePayload(payload) }
    case 'interrupt_requested': {
      const object = parseRequiredObject(payload)
      return {
        kind: 'interrupt_requested',
        payload: {
          interruptId: readRequiredString(object, 'interrupt_id'),
          provider: readRequiredString(object, 'provider'),
          kind: readRequiredString(object, 'kind'),
          options: readInterruptOptions(object),
          payload: readOptionalObject(object, 'payload') ?? {},
        },
      }
    }
    case 'interrupt_resolved': {
      const object = parseRequiredObject(payload)
      return {
        kind: 'interrupt_resolved',
        payload: {
          interruptId: readRequiredString(object, 'interrupt_id'),
          decision: readOptionalString(object, 'decision'),
        },
      }
    }
    case 'diff_updated': {
      const object = parseRequiredObject(payload)
      return {
        kind: 'diff_updated',
        payload: {
          threadId: readRequiredString(object, 'thread_id'),
          turnId: readRequiredString(object, 'turn_id'),
          diff: readRequiredString(object, 'diff'),
          entryId: readOptionalString(object, 'entry_id'),
        },
      }
    }
    case 'reasoning_updated': {
      const object = parseRequiredObject(payload)
      return {
        kind: 'reasoning_updated',
        payload: {
          threadId: readRequiredString(object, 'thread_id'),
          turnId: readRequiredString(object, 'turn_id'),
          itemId: readRequiredString(object, 'item_id'),
          kind: readRequiredString(object, 'kind'),
          delta: readOptionalString(object, 'delta'),
          summaryIndex: readOptionalNumber(object, 'summary_index'),
          contentIndex: readOptionalNumber(object, 'content_index'),
          entryId: readOptionalString(object, 'entry_id'),
        },
      }
    }
    case 'interrupted': {
      const object = parseRequiredObject(payload)
      return {
        kind: 'interrupted',
        payload: {
          conversationId: readOptionalString(object, 'conversation_id'),
          turnId: readOptionalString(object, 'turn_id'),
          message: readRequiredString(object, 'message'),
          reason: readOptionalString(object, 'reason'),
        },
      }
    }
    case 'turn_done': {
      const object = parseRequiredObject(payload)
      return {
        kind: 'turn_done',
        payload: {
          conversationId: readRequiredString(object, 'conversation_id'),
          turnId: readRequiredString(object, 'turn_id'),
          costUSD: readOptionalNumber(object, 'cost_usd'),
        },
      }
    }
    case 'error':
      return { kind: 'error', payload: parseErrorPayload(payload) }
    default:
      return null
  }
}

export function parseRawProjectConversationMuxFrame(
  frame: Pick<SSEFrame, 'event' | 'data'>,
): ProjectConversationMuxFrameParseResult {
  try {
    const raw = parseJSONObject(frame.data)
    if (raw == null) {
      return {
        ok: false,
        error: new Error('project conversation mux frame must contain JSON data'),
      }
    }

    const object = parseRequiredObject(raw)
    const event = parseProjectConversationStreamPayload(frame.event, object.payload)
    if (event == null) {
      return {
        ok: false,
        error: new Error(`project conversation mux event ${frame.event} is unsupported`),
      }
    }

    return {
      ok: true,
      value: {
        conversationId: readRequiredString(object, 'conversation_id'),
        sentAt: readRequiredString(object, 'sent_at'),
        event,
      },
    }
  } catch (error) {
    return {
      ok: false,
      error:
        error instanceof Error ? error : new Error('project conversation mux frame parsing failed'),
    }
  }
}

function parseProjectConversationWorkspaceDiff(value: unknown): ProjectConversationWorkspaceDiff {
  const object = parseRequiredObject(value)
  return {
    conversationId: readRequiredString(object, 'conversation_id'),
    workspacePath: readRequiredString(object, 'workspace_path'),
    dirty: readRequiredBoolean(object, 'dirty'),
    reposChanged: readRequiredNumber(object, 'repos_changed'),
    filesChanged: readRequiredNumber(object, 'files_changed'),
    added: readRequiredNumber(object, 'added'),
    removed: readRequiredNumber(object, 'removed'),
    repos: readProjectConversationWorkspaceDiffRepos(object),
    syncPrompt: parseProjectConversationWorkspaceSyncPrompt(
      readOptionalObject(object, 'sync_prompt'),
    ),
  }
}

function parseProjectConversationWorkspaceMetadata(
  value: unknown,
): ProjectConversationWorkspaceMetadata {
  const object = parseRequiredObject(value)
  const repos = Array.isArray(object.repos)
    ? object.repos.map((item) => {
        const repo = parseRequiredObject(item)
        return {
          name: readRequiredString(repo, 'name'),
          path: readRequiredString(repo, 'path'),
          branch: readRequiredString(repo, 'branch'),
          currentRef: parseProjectConversationWorkspaceCurrentRef(
            readOptionalObject(repo, 'current_ref'),
            readRequiredString(repo, 'branch'),
            readRequiredString(repo, 'head_commit'),
            readRequiredString(repo, 'head_summary'),
          ),
          headCommit: readRequiredString(repo, 'head_commit'),
          headSummary: readRequiredString(repo, 'head_summary'),
          dirty: readRequiredBoolean(repo, 'dirty'),
          filesChanged: readRequiredNumber(repo, 'files_changed'),
          added: readRequiredNumber(repo, 'added'),
          removed: readRequiredNumber(repo, 'removed'),
        } satisfies ProjectConversationWorkspaceRepoMetadata
      })
    : []

  return {
    conversationId: readRequiredString(object, 'conversation_id'),
    available: readRequiredBoolean(object, 'available'),
    workspacePath: readOptionalString(object, 'workspace_path') ?? '',
    repos,
    syncPrompt: parseProjectConversationWorkspaceSyncPrompt(
      readOptionalObject(object, 'sync_prompt'),
    ),
  }
}

function parseProjectConversationWorkspaceCurrentRef(
  value: Record<string, unknown> | undefined,
  fallbackDisplayName = '',
  fallbackCommitId = '',
  fallbackSubject = '',
): ProjectConversationWorkspaceCurrentRef {
  if (!value) {
    const kind: ProjectConversationWorkspaceCurrentRefKind = fallbackDisplayName.startsWith(
      'detached',
    )
      ? 'detached'
      : 'branch'
    return {
      kind,
      displayName: fallbackDisplayName,
      cacheKey:
        kind === 'branch' ? `branch:${fallbackDisplayName}` : `detached:${fallbackCommitId || ''}`,
      branchName: kind === 'branch' ? fallbackDisplayName : '',
      branchFullName: kind === 'branch' ? `refs/heads/${fallbackDisplayName}` : '',
      commitId: fallbackCommitId,
      shortCommitId: fallbackCommitId.slice(0, 12),
      subject: fallbackSubject,
    }
  }
  const kind = readRequiredString(value, 'kind')
  if (kind !== 'branch' && kind !== 'detached') {
    throw new Error(`project conversation workspace current ref ${kind} is unsupported`)
  }
  return {
    kind,
    displayName: readRequiredString(value, 'display_name'),
    cacheKey: readRequiredString(value, 'cache_key'),
    branchName: readOptionalString(value, 'branch_name') ?? '',
    branchFullName: readOptionalString(value, 'branch_full_name') ?? '',
    commitId: readOptionalString(value, 'commit_id') ?? '',
    shortCommitId: readOptionalString(value, 'short_commit_id') ?? '',
    subject: readOptionalString(value, 'subject') ?? '',
  }
}

function parseProjectConversationWorkspaceBranchRef(
  value: unknown,
): ProjectConversationWorkspaceBranchRef {
  const object = parseRequiredObject(value)
  const scope = readRequiredString(object, 'scope')
  if (scope !== 'local_branch' && scope !== 'remote_tracking_branch') {
    throw new Error(`project conversation workspace branch scope ${scope} is unsupported`)
  }
  return {
    name: readRequiredString(object, 'name'),
    fullName: readRequiredString(object, 'full_name'),
    scope,
    current: readRequiredBoolean(object, 'current'),
    commitId: readOptionalString(object, 'commit_id') ?? '',
    shortCommitId: readOptionalString(object, 'short_commit_id') ?? '',
    subject: readOptionalString(object, 'subject') ?? '',
    upstreamName: readOptionalString(object, 'upstream_name') ?? '',
    ahead: readRequiredNumber(object, 'ahead'),
    behind: readRequiredNumber(object, 'behind'),
    suggestedLocalBranchName: readOptionalString(object, 'suggested_local_branch_name') ?? '',
  }
}

function parseProjectConversationWorkspaceRepoRefs(
  value: unknown,
): ProjectConversationWorkspaceRepoRefs {
  const object = parseRequiredObject(value)
  return {
    conversationId: readRequiredString(object, 'conversation_id'),
    repoPath: readRequiredString(object, 'repo_path'),
    currentRef: parseProjectConversationWorkspaceCurrentRef(
      readOptionalObject(object, 'current_ref'),
    ),
    localBranches: Array.isArray(object.local_branches)
      ? object.local_branches.map((item) => parseProjectConversationWorkspaceBranchRef(item))
      : [],
    remoteBranches: Array.isArray(object.remote_branches)
      ? object.remote_branches.map((item) => parseProjectConversationWorkspaceBranchRef(item))
      : [],
  }
}

function parseProjectConversationWorkspaceGitRefLabel(
  value: unknown,
): ProjectConversationWorkspaceGitRefLabel {
  const object = parseRequiredObject(value)
  const scope = readRequiredString(object, 'scope')
  if (scope !== 'head' && scope !== 'local_branch' && scope !== 'remote_tracking_branch') {
    throw new Error(`project conversation workspace git label scope ${scope} is unsupported`)
  }
  return {
    name: readRequiredString(object, 'name'),
    fullName: readRequiredString(object, 'full_name'),
    scope,
    current: readRequiredBoolean(object, 'current'),
  }
}

function parseProjectConversationWorkspaceGitGraph(
  value: unknown,
): ProjectConversationWorkspaceGitGraph {
  const object = parseRequiredObject(value)
  const commits = Array.isArray(object.commits)
    ? object.commits.map((item) => {
        const commit = parseRequiredObject(item)
        return {
          commitId: readRequiredString(commit, 'commit_id'),
          shortCommitId: readRequiredString(commit, 'short_commit_id'),
          parentIds: Array.isArray(commit.parent_ids)
            ? commit.parent_ids.map((parent) => String(parent))
            : [],
          subject: readRequiredString(commit, 'subject'),
          authorName: readRequiredString(commit, 'author_name'),
          authoredAt: readRequiredString(commit, 'authored_at'),
          labels: Array.isArray(commit.labels)
            ? commit.labels.map((label) => parseProjectConversationWorkspaceGitRefLabel(label))
            : [],
          head: readRequiredBoolean(commit, 'head'),
        } satisfies ProjectConversationWorkspaceGitGraphCommit
      })
    : []
  return {
    conversationId: readRequiredString(object, 'conversation_id'),
    repoPath: readRequiredString(object, 'repo_path'),
    limit: readRequiredNumber(object, 'limit'),
    commits,
  }
}

function parseProjectConversationWorkspaceCheckoutResult(
  value: unknown,
): ProjectConversationWorkspaceCheckoutResult {
  const object = parseRequiredObject(value)
  return {
    conversationId: readRequiredString(object, 'conversation_id'),
    repoPath: readRequiredString(object, 'repo_path'),
    currentRef: parseProjectConversationWorkspaceCurrentRef(
      readOptionalObject(object, 'current_ref'),
    ),
    createdLocalBranch: readOptionalString(object, 'created_local_branch') ?? '',
  }
}

function parseProjectConversationWorkspaceSyncPrompt(
  value?: Record<string, unknown>,
): ProjectConversationWorkspaceSyncPrompt | undefined {
  if (!value) {
    return undefined
  }
  const reason = readRequiredString(value, 'reason')
  if (reason !== 'repo_binding_changed' && reason !== 'repo_missing') {
    throw new Error(`project conversation workspace sync reason ${reason} is unsupported`)
  }
  const missingRepos = Array.isArray(value.missing_repos)
    ? value.missing_repos.map((item) => {
        const repo = parseRequiredObject(item)
        return {
          name: readRequiredString(repo, 'name'),
          path: readRequiredString(repo, 'path'),
        } satisfies ProjectConversationWorkspaceMissingRepo
      })
    : []
  return {
    reason,
    missingRepos,
  }
}

function parseProjectConversationWorkspaceTree(value: unknown): ProjectConversationWorkspaceTree {
  const object = parseRequiredObject(value)
  const entries = Array.isArray(object.entries)
    ? object.entries.map((item) => {
        const entry = parseRequiredObject(item)
        const kind = readRequiredString(entry, 'kind')
        if (kind !== 'directory' && kind !== 'file') {
          throw new Error(`project conversation workspace tree kind ${kind} is unsupported`)
        }
        return {
          path: readRequiredString(entry, 'path'),
          name: readRequiredString(entry, 'name'),
          kind,
          sizeBytes: readRequiredNumber(entry, 'size_bytes'),
        } satisfies ProjectConversationWorkspaceTreeEntry
      })
    : []

  return {
    conversationId: readRequiredString(object, 'conversation_id'),
    repoPath: readRequiredString(object, 'repo_path'),
    path: typeof object.path === 'string' ? object.path : '',
    entries,
  }
}

function parseProjectConversationWorkspaceSearch(
  value: unknown,
): ProjectConversationWorkspaceSearch {
  const object = parseRequiredObject(value)
  const results = Array.isArray(object.results)
    ? object.results.map((item) => {
        const entry = parseRequiredObject(item)
        return {
          path: readRequiredString(entry, 'path'),
          name: readRequiredString(entry, 'name'),
        } satisfies ProjectConversationWorkspaceSearchResult
      })
    : []

  return {
    conversationId: readRequiredString(object, 'conversation_id'),
    repoPath: readRequiredString(object, 'repo_path'),
    query: readRequiredString(object, 'query'),
    truncated: readRequiredBoolean(object, 'truncated'),
    results,
  }
}

function parseProjectConversationWorkspaceFilePreview(
  value: unknown,
): ProjectConversationWorkspaceFilePreview {
  const object = parseRequiredObject(value)
  const previewKind = readRequiredString(object, 'preview_kind')
  if (previewKind !== 'text' && previewKind !== 'binary') {
    throw new Error(`project conversation workspace preview kind ${previewKind} is unsupported`)
  }
  return {
    conversationId: readRequiredString(object, 'conversation_id'),
    repoPath: readRequiredString(object, 'repo_path'),
    path: readRequiredString(object, 'path'),
    sizeBytes: readRequiredNumber(object, 'size_bytes'),
    mediaType: readRequiredString(object, 'media_type'),
    previewKind,
    truncated: readRequiredBoolean(object, 'truncated'),
    content: readOptionalString(object, 'content') ?? '',
    revision: readRequiredString(object, 'revision'),
    writable: readRequiredBoolean(object, 'writable'),
    readOnlyReason: readOptionalString(object, 'read_only_reason') ?? '',
    encoding: readRequiredString(object, 'encoding') as 'utf-8',
    lineEnding: readRequiredString(object, 'line_ending') as 'lf' | 'crlf',
  }
}

function parseProjectConversationWorkspaceFileSaved(
  value: unknown,
): ProjectConversationWorkspaceFileSaved {
  const object = parseRequiredObject(value)
  return {
    conversationId: readRequiredString(object, 'conversation_id'),
    repoPath: readRequiredString(object, 'repo_path'),
    path: readRequiredString(object, 'path'),
    revision: readRequiredString(object, 'revision'),
    sizeBytes: readRequiredNumber(object, 'size_bytes'),
    encoding: readRequiredString(object, 'encoding') as 'utf-8',
    lineEnding: readRequiredString(object, 'line_ending') as 'lf' | 'crlf',
  }
}

function parseProjectConversationWorkspaceFileCreated(
  value: unknown,
): ProjectConversationWorkspaceFileCreated {
  const object = parseRequiredObject(value)
  return {
    conversationId: readRequiredString(object, 'conversation_id'),
    repoPath: readRequiredString(object, 'repo_path'),
    path: readRequiredString(object, 'path'),
    revision: readRequiredString(object, 'revision'),
    sizeBytes: readRequiredNumber(object, 'size_bytes'),
    encoding: readRequiredString(object, 'encoding') as 'utf-8',
    lineEnding: readRequiredString(object, 'line_ending') as 'lf' | 'crlf',
  }
}

function parseProjectConversationWorkspaceFileRenamed(
  value: unknown,
): ProjectConversationWorkspaceFileRenamed {
  const object = parseRequiredObject(value)
  return {
    conversationId: readRequiredString(object, 'conversation_id'),
    repoPath: readRequiredString(object, 'repo_path'),
    fromPath: readRequiredString(object, 'from_path'),
    toPath: readRequiredString(object, 'to_path'),
  }
}

function parseProjectConversationWorkspaceFileDeleted(
  value: unknown,
): ProjectConversationWorkspaceFileDeleted {
  const object = parseRequiredObject(value)
  return {
    conversationId: readRequiredString(object, 'conversation_id'),
    repoPath: readRequiredString(object, 'repo_path'),
    path: readRequiredString(object, 'path'),
  }
}

function parseProjectConversationWorkspaceFilePatch(
  value: unknown,
): ProjectConversationWorkspaceFilePatch {
  const object = parseRequiredObject(value)
  const status = readRequiredString(object, 'status')
  if (!isProjectConversationWorkspaceFileStatus(status)) {
    throw new Error(`project conversation workspace file status ${status} is unsupported`)
  }
  const diffKind = readRequiredString(object, 'diff_kind')
  if (diffKind !== 'none' && diffKind !== 'text' && diffKind !== 'binary') {
    throw new Error(`project conversation workspace diff kind ${diffKind} is unsupported`)
  }
  return {
    conversationId: readRequiredString(object, 'conversation_id'),
    repoPath: readRequiredString(object, 'repo_path'),
    path: readRequiredString(object, 'path'),
    status,
    diffKind,
    truncated: readRequiredBoolean(object, 'truncated'),
    diff: readOptionalString(object, 'diff') ?? '',
  }
}

function parseProjectConversationTerminalSession(
  value: unknown,
): ProjectConversationTerminalSession {
  const object = parseRequiredObject(value)
  const mode = readRequiredString(object, 'mode')
  if (mode !== 'shell') {
    throw new Error(`project conversation terminal mode ${mode} is unsupported`)
  }
  return {
    id: readRequiredString(object, 'id'),
    mode,
    cwd: readRequiredString(object, 'cwd'),
    wsPath: readRequiredString(object, 'ws_path'),
    attachToken: readRequiredString(object, 'attach_token'),
  }
}

function readProjectConversationWorkspaceDiffRepos(
  object: Record<string, unknown>,
): ProjectConversationWorkspaceDiffRepo[] {
  const value = object.repos
  if (!Array.isArray(value)) {
    return []
  }

  return value.map((item) => {
    const repo = parseRequiredObject(item)
    return {
      name: readRequiredString(repo, 'name'),
      path: readRequiredString(repo, 'path'),
      branch: readRequiredString(repo, 'branch'),
      dirty: readRequiredBoolean(repo, 'dirty'),
      filesChanged: readRequiredNumber(repo, 'files_changed'),
      added: readRequiredNumber(repo, 'added'),
      removed: readRequiredNumber(repo, 'removed'),
      files: readProjectConversationWorkspaceDiffFiles(repo),
    }
  })
}

function readProjectConversationWorkspaceDiffFiles(
  object: Record<string, unknown>,
): ProjectConversationWorkspaceDiffFile[] {
  const value = object.files
  if (!Array.isArray(value)) {
    return []
  }

  return value.map((item) => {
    const file = parseRequiredObject(item)
    const status = readRequiredString(file, 'status')
    if (!isProjectConversationWorkspaceFileStatus(status)) {
      throw new Error(`project conversation workspace file status ${status} is unsupported`)
    }
    return {
      path: readRequiredString(file, 'path'),
      oldPath: readOptionalString(file, 'old_path') ?? '',
      status,
      staged: readOptionalBoolean(file, 'staged') ?? false,
      unstaged: readOptionalBoolean(file, 'unstaged') ?? false,
      added: readRequiredNumber(file, 'added'),
      removed: readRequiredNumber(file, 'removed'),
    }
  })
}

function isProjectConversationWorkspaceFileStatus(
  value: string,
): value is ProjectConversationWorkspaceFileStatus {
  return (
    value === 'modified' ||
    value === 'added' ||
    value === 'deleted' ||
    value === 'renamed' ||
    value === 'untracked'
  )
}

function parseMessagePayload(payload: unknown): ChatMessagePayload {
  const object = parseRequiredObject(payload)
  const type = readRequiredString(object, 'type')

  if (type === 'text') {
    return {
      type,
      content: readRequiredString(object, 'content'),
    }
  }

  if (type === 'diff') {
    return {
      type,
      entryId: readOptionalString(object, 'entry_id'),
      file: readRequiredString(object, 'file'),
      hunks: readDiffHunks(object),
    }
  }

  if (type === 'bundle_diff') {
    return {
      type,
      entryId: readOptionalString(object, 'entry_id'),
      files: readBundleDiffFiles(object),
    }
  }

  return {
    type,
    raw: readOptionalObject(object, 'raw') ?? object,
  }
}

function parseSessionPayload(payload: unknown): ChatSessionPayload {
  const object = parseRequiredObject(payload)
  return {
    sessionId: readRequiredString(object, 'session_id'),
  }
}

function parseDonePayload(payload: unknown): ChatDonePayload {
  const object = parseRequiredObject(payload)
  return {
    sessionId: readRequiredString(object, 'session_id'),
    turnsUsed: readRequiredNumber(object, 'turns_used'),
    turnsRemaining: readOptionalNumber(object, 'turns_remaining'),
    costUSD: readOptionalNumber(object, 'cost_usd'),
  }
}

function parseErrorPayload(payload: unknown): ChatErrorPayload {
  const object = parseRequiredObject(payload)
  return {
    message: readRequiredString(object, 'message'),
  }
}

function parseProjectConversation(value: unknown): ProjectConversation {
  const object = parseRequiredObject(value)
  const createdAt = readOptionalString(object, 'created_at') ?? ''
  const updatedAt = readOptionalString(object, 'updated_at') ?? createdAt

  return {
    id: readRequiredString(object, 'id'),
    projectId: readOptionalString(object, 'project_id') ?? '',
    userId: readOptionalString(object, 'user_id') ?? '',
    source: 'project_sidebar',
    providerId: readOptionalString(object, 'provider_id') ?? '',
    title: readOptionalString(object, 'title') ?? '',
    providerAnchorKind: readProviderAnchorKind(object),
    providerAnchorId:
      readOptionalString(object, 'provider_anchor_id') ??
      readOptionalString(object, 'provider_thread_id'),
    providerTurnId:
      readOptionalString(object, 'provider_turn_id') ?? readOptionalString(object, 'last_turn_id'),
    providerTurnSupported: readOptionalBoolean(object, 'provider_turn_supported'),
    providerStatus:
      readOptionalString(object, 'provider_status') ??
      readOptionalString(object, 'provider_thread_status'),
    providerActiveFlags:
      readOptionalStringList(object, 'provider_active_flags').length > 0
        ? readOptionalStringList(object, 'provider_active_flags')
        : readOptionalStringList(object, 'provider_thread_active_flags'),
    status: readOptionalString(object, 'status') ?? '',
    rollingSummary: readOptionalString(object, 'rolling_summary') ?? '',
    lastActivityAt: readOptionalString(object, 'last_activity_at') ?? updatedAt ?? createdAt,
    createdAt,
    updatedAt,
    runtimePrincipal: parseProjectConversationRuntimePrincipal(
      readOptionalObject(object, 'runtime_principal'),
    ),
  }
}

function parseProjectConversationRuntimePrincipal(
  value?: Record<string, unknown> | null,
): ProjectConversationRuntimePrincipal | undefined {
  if (value == null) {
    return undefined
  }

  return {
    id: readOptionalString(value, 'id'),
    name: readOptionalString(value, 'name'),
    status: readOptionalString(value, 'status'),
    runtimeState: readOptionalString(value, 'runtime_state'),
    currentSessionId: readOptionalString(value, 'current_session_id'),
    currentWorkspacePath: readOptionalString(value, 'current_workspace_path'),
    currentRunId: readOptionalString(value, 'current_run_id'),
    lastHeartbeatAt: readOptionalString(value, 'last_heartbeat_at'),
    currentStepStatus: readOptionalString(value, 'current_step_status'),
    currentStepSummary: readOptionalString(value, 'current_step_summary'),
    currentStepChangedAt: readOptionalString(value, 'current_step_changed_at'),
  }
}

function parseProjectConversationEntry(value: unknown): ProjectConversationEntry {
  const object = parseRequiredObject(value)
  return {
    id: readRequiredString(object, 'id'),
    conversationId: readRequiredString(object, 'conversation_id'),
    turnId: readOptionalString(object, 'turn_id'),
    seq: readRequiredNumber(object, 'seq'),
    kind: readRequiredString(object, 'kind'),
    payload: readOptionalObject(object, 'payload') ?? {},
    createdAt: readOptionalString(object, 'created_at') ?? '',
  }
}

function parseJSONObject(raw: string): unknown | null {
  try {
    return JSON.parse(raw) as unknown
  } catch {
    return null
  }
}

function parseRequiredObject(value: unknown): Record<string, unknown> {
  if (value == null || typeof value !== 'object' || Array.isArray(value)) {
    throw new Error('chat stream payload must be an object')
  }

  return value as Record<string, unknown>
}

function readRequiredString(object: Record<string, unknown>, key: string): string {
  const value = object[key]
  if (typeof value !== 'string' || value.trim() === '') {
    throw new Error(`chat stream payload field ${key} must be a non-empty string`)
  }
  return value
}

function readOptionalString(object: Record<string, unknown>, key: string): string | undefined {
  const value = object[key]
  return typeof value === 'string' && value.trim() !== '' ? value : undefined
}

function readRequiredNumber(object: Record<string, unknown>, key: string): number {
  const value = object[key]
  if (typeof value !== 'number' || Number.isNaN(value)) {
    throw new Error(`chat stream payload field ${key} must be a number`)
  }
  return value
}

function readRequiredBoolean(object: Record<string, unknown>, key: string): boolean {
  const value = object[key]
  if (typeof value !== 'boolean') {
    throw new Error(`chat stream payload field ${key} must be a boolean`)
  }
  return value
}

function readOptionalNumber(object: Record<string, unknown>, key: string): number | undefined {
  const value = object[key]
  return typeof value === 'number' && !Number.isNaN(value) ? value : undefined
}

function readOptionalBoolean(object: Record<string, unknown>, key: string): boolean | undefined {
  const value = object[key]
  return typeof value === 'boolean' ? value : undefined
}

function readOptionalObject(
  object: Record<string, unknown>,
  key: string,
): Record<string, unknown> | undefined {
  const value = object[key]
  return value != null && typeof value === 'object' && !Array.isArray(value)
    ? (value as Record<string, unknown>)
    : undefined
}

function readOptionalStringList(object: Record<string, unknown>, key: string): string[] {
  const value = object[key]
  if (!Array.isArray(value)) {
    return []
  }
  return value.filter((item): item is string => typeof item === 'string' && item.trim() !== '')
}

function readProviderAnchorKind(
  object: Record<string, unknown>,
): ProjectConversation['providerAnchorKind'] {
  const value = readOptionalString(object, 'provider_anchor_kind')
  return value === 'thread' || value === 'session' ? value : undefined
}

function readInterruptOptions(
  object: Record<string, unknown>,
): ProjectConversationInterruptOption[] {
  const value = object.options
  if (!Array.isArray(value)) {
    return []
  }

  return value.map((item) => {
    const option = parseRequiredObject(item)
    return {
      id: readRequiredString(option, 'id'),
      label: readRequiredString(option, 'label'),
    }
  })
}

function readDiffHunks(object: Record<string, unknown>): ChatDiffHunk[] {
  const hunks = object.hunks
  if (!Array.isArray(hunks) || hunks.length === 0) {
    throw new Error('chat stream diff hunks must be a non-empty array')
  }

  return hunks.map((hunk, index) => parseDiffHunk(hunk, index))
}

function readBundleDiffFiles(object: Record<string, unknown>): ChatBundleDiffFile[] {
  const files = object.files
  if (!Array.isArray(files) || files.length === 0) {
    throw new Error('chat stream bundle_diff files must be a non-empty array')
  }

  const seen = new Set<string>()
  return files.map((item, index) => {
    const fileObject = parseRequiredObject(item)
    const file = readRequiredString(fileObject, 'file')
    if (seen.has(file)) {
      throw new Error(`chat stream bundle_diff file ${index} is duplicated`)
    }
    seen.add(file)
    return {
      file,
      hunks: readDiffHunks(fileObject),
    }
  })
}

function parseDiffHunk(value: unknown, index: number): ChatDiffHunk {
  const object = parseRequiredObject(value)
  const oldStart = readRequiredNumber(object, 'old_start')
  const oldLines = readRequiredNumber(object, 'old_lines')
  const newStart = readRequiredNumber(object, 'new_start')
  const newLines = readRequiredNumber(object, 'new_lines')
  const lines = readDiffLines(object, index)

  if (!Number.isInteger(oldStart) || oldStart < 1) {
    throw new Error(`chat stream diff hunk ${index} old_start must be a positive integer`)
  }
  if (!Number.isInteger(newStart) || newStart < 1) {
    throw new Error(`chat stream diff hunk ${index} new_start must be a positive integer`)
  }
  if (!Number.isInteger(oldLines) || oldLines < 0) {
    throw new Error(`chat stream diff hunk ${index} old_lines must be a non-negative integer`)
  }
  if (!Number.isInteger(newLines) || newLines < 0) {
    throw new Error(`chat stream diff hunk ${index} new_lines must be a non-negative integer`)
  }

  return {
    oldStart,
    oldLines,
    newStart,
    newLines,
    lines,
  }
}

function readDiffLines(object: Record<string, unknown>, index: number): ChatDiffLine[] {
  const lines = object.lines
  if (!Array.isArray(lines) || lines.length === 0) {
    throw new Error(`chat stream diff hunk ${index} lines must be a non-empty array`)
  }

  return lines.map((line, lineIndex) => parseDiffLine(line, index, lineIndex))
}

function parseDiffLine(value: unknown, hunkIndex: number, lineIndex: number): ChatDiffLine {
  const object = parseRequiredObject(value)
  const op = readRequiredString(object, 'op')
  if (!isDiffLineOp(op)) {
    throw new Error(`chat stream diff hunk ${hunkIndex} line ${lineIndex} op is unsupported`)
  }

  return {
    op,
    text: readRequiredString(object, 'text'),
  }
}

function isDiffLineOp(value: string): value is ChatDiffLineOp {
  return value === 'context' || value === 'add' || value === 'remove'
}

async function fetchJSON<T>(
  path: string,
  options?: {
    method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'
    params?: Record<string, string | undefined>
    body?: unknown
  },
) {
  const url = new URL(path, window.location.origin)
  for (const [key, value] of Object.entries(options?.params ?? {})) {
    if (value) {
      url.searchParams.set(key, value)
    }
  }

  const method = options?.method ?? 'GET'
  const headers = buildRequestHeaders(method, {
    ...(options?.body ? { 'Content-Type': 'application/json' } : {}),
  })

  const response = await fetch(url.toString(), {
    method,
    headers,
    body: options?.body ? JSON.stringify(options.body) : undefined,
    credentials: 'same-origin',
  })
  if (!response.ok) {
    let detail = response.statusText
    let code: string | undefined
    let details: unknown
    try {
      const contentType = response.headers.get('content-type') ?? ''
      if (contentType.includes('application/json')) {
        const payload = (await response.json()) as {
          message?: string
          detail?: string
          error?: string
          code?: string
          details?: unknown
        }
        detail =
          payload.message || payload.detail || payload.error || payload.code || response.statusText
        code = payload.code
        details = payload.details
      } else {
        detail = await response.text()
      }
    } catch {
      detail = response.statusText
    }
    throw new ApiError(response.status, detail, code, details)
  }
  if (response.status === 204) {
    return undefined as T
  }
  return response.json() as Promise<T>
}
