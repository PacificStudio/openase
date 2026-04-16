import { DEFAULT_PROVIDER_ID, PROJECT_ID, nowIso } from './constants'
import {
  asBoolean,
  asObject,
  asString,
  clone,
  findById,
  jsonResponse,
  notFound,
  readBody,
} from './helpers'
import { getMockState } from './store'
import {
  nextProjectConversationSeq,
  projectConversationMuxStreamResponse,
  projectConversationStreamResponse,
  queueOrBroadcastProjectConversationEvent,
  shiftedIso,
  streamResponse,
} from './streams'
import {
  buildMockProjectConversationReply,
  buildMockProjectConversationWorkspaceDiff,
} from './ticket-data'

export async function handleChatRoutes(request: Request, segments: string[]) {
  const state = getMockState()
  if (segments.length === 1) {
    if (request.method === 'POST') {
      return streamResponse()
    }
    return notFound('Mock chat route not found.')
  }

  if (segments[1] !== 'conversations') {
    if (
      segments[1] === 'projects' &&
      segments.length === 5 &&
      segments[3] === 'conversations' &&
      segments[4] === 'stream' &&
      request.method === 'GET'
    ) {
      if (segments[2] !== PROJECT_ID) {
        return notFound('Project not found.')
      }
      return projectConversationMuxStreamResponse(segments[2])
    }
    return notFound('Mock chat route not found.')
  }

  if (segments.length === 2 && request.method === 'GET') {
    const search = new URL(request.url).searchParams
    const projectId = search.get('project_id') ?? PROJECT_ID
    const providerId = search.get('provider_id')
    return jsonResponse({
      conversations: clone(
        state.projectConversations.filter((conversation) => {
          if (conversation.project_id !== projectId) {
            return false
          }
          if (providerId && conversation.provider_id !== providerId) {
            return false
          }
          return true
        }),
      ),
    })
  }

  if (segments.length === 2 && request.method === 'POST') {
    const body = await readBody<Record<string, unknown>>(request)
    const providerId = asString(body.provider_id) ?? DEFAULT_PROVIDER_ID
    const conversation = {
      id: `conversation-e2e-${++state.counters.projectConversation}`,
      project_id: PROJECT_ID,
      user_id: 'chat-user-e2e',
      source: 'project_sidebar',
      provider_id: providerId,
      status: 'active',
      rolling_summary: '',
      last_activity_at: nowIso,
      created_at: nowIso,
      updated_at: nowIso,
    }
    state.projectConversations.unshift(conversation)
    return jsonResponse(
      {
        conversation: clone(conversation),
      },
      201,
    )
  }

  if (segments.length === 3 && request.method === 'GET') {
    const conversation = findById(state.projectConversations, segments[2])
    if (!conversation) {
      return notFound('Project conversation not found.')
    }
    return jsonResponse({ conversation: clone(conversation) })
  }

  if (segments[3] === 'workspace-diff' && request.method === 'GET') {
    const conversation = findById(state.projectConversations, segments[2])
    if (!conversation) {
      return notFound('Project conversation not found.')
    }
    return jsonResponse({
      workspace_diff: buildMockProjectConversationWorkspaceDiff(segments[2]),
    })
  }

  if (segments[3] === 'entries' && request.method === 'GET') {
    return jsonResponse({
      entries: clone(
        state.projectConversationEntries.filter((entry) => entry.conversation_id === segments[2]),
      ),
    })
  }

  if (segments[3] === 'stream' && request.method === 'GET') {
    return projectConversationStreamResponse(segments[2])
  }

  if (segments[3] === 'turns' && request.method === 'POST') {
    const conversation = findById(state.projectConversations, segments[2])
    if (!conversation) {
      return notFound('Project conversation not found.')
    }

    const body = await readBody<Record<string, unknown>>(request)
    const message = asString(body.message) ?? ''
    const focus = asObject(body.focus)
    const ticketFocus =
      asString(focus?.kind) === 'ticket'
        ? {
            identifier: asString(focus?.ticket_identifier) ?? 'ASE-unknown',
            status: asString(focus?.ticket_status) ?? '',
            retryPaused: asBoolean(focus?.ticket_retry_paused) ?? false,
            pauseReason: asString(focus?.ticket_pause_reason) ?? '',
            repoScopes: Array.isArray(focus?.ticket_repo_scopes) ? focus.ticket_repo_scopes : [],
            hookHistory: Array.isArray(focus?.ticket_hook_history) ? focus.ticket_hook_history : [],
            currentRun: asObject(focus?.ticket_current_run),
          }
        : null
    const turnIndex = ++state.counters.projectConversationTurn
    const turnId = `turn-e2e-${turnIndex}`
    const userEntry = {
      id: `entry-e2e-${++state.counters.projectConversationEntry}`,
      conversation_id: segments[2],
      turn_id: turnId,
      seq: nextProjectConversationSeq(segments[2]),
      kind: 'user_message',
      payload: { content: message },
      created_at: shiftedIso(turnIndex),
    }
    const assistantPayload = {
      kind: 'assistant_message',
      payload: {
        content: buildMockProjectConversationReply(message, ticketFocus),
      },
    }
    const assistantEntry = {
      id: `entry-e2e-${++state.counters.projectConversationEntry}`,
      conversation_id: segments[2],
      turn_id: turnId,
      seq: userEntry.seq + 1,
      kind: assistantPayload.kind,
      payload: assistantPayload.payload,
      created_at: shiftedIso(turnIndex + 1),
    }
    state.projectConversationEntries.push(userEntry, assistantEntry)
    conversation.last_activity_at = assistantEntry.created_at
    conversation.updated_at = assistantEntry.created_at
    conversation.rolling_summary = message

    const sessionSentAt = shiftedIso(turnIndex)
    queueOrBroadcastProjectConversationEvent(
      segments[2],
      'session',
      {
        conversation_id: segments[2],
        runtime_state: 'active',
      },
      sessionSentAt,
    )
    setTimeout(() => {
      queueOrBroadcastProjectConversationEvent(
        segments[2],
        'message',
        {
          type: 'text',
          content: String(assistantEntry.payload.content ?? ''),
        },
        shiftedIso(turnIndex + 1),
      )
    }, 25)
    setTimeout(() => {
      queueOrBroadcastProjectConversationEvent(
        segments[2],
        'turn_done',
        {
          conversation_id: segments[2],
          turn_id: turnId,
          cost_usd: 0.01,
        },
        shiftedIso(turnIndex + 2),
      )
    }, 50)

    return jsonResponse(
      {
        turn: {
          id: turnId,
          turn_index: turnIndex,
          status: 'started',
        },
      },
      202,
    )
  }

  return notFound('Mock chat route not found.')
}
