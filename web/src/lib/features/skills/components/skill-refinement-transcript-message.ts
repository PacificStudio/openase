import type { SkillRefinementMessagePayload } from '$lib/api/skill-refinement'
import {
  mapProjectConversationTaskEntry,
  type ProjectConversationTranscriptEntry,
} from '$lib/features/chat'

export type SkillRefinementMessageEntry =
  | ProjectConversationTranscriptEntry
  | { kind: 'text'; content: string }

export function mapSkillRefinementMessageEvent(
  payload: SkillRefinementMessagePayload,
  id: string,
): SkillRefinementMessageEntry | null {
  if (payload.type === 'text') {
    const content = typeof payload.content === 'string' ? payload.content.trim() : ''
    if (!content || looksLikeSkillRefinementResult(content)) return null
    return { kind: 'text', content }
  }

  if (payload.type === 'diff' && isRecord(payload)) {
    return {
      id,
      kind: 'diff',
      role: 'assistant',
      diff: {
        type: 'diff',
        file: typeof payload.file === 'string' ? payload.file : '',
        hunks: Array.isArray(payload.hunks) ? (payload.hunks as never[]) : [],
      },
    } satisfies ProjectConversationTranscriptEntry
  }

  return mapProjectConversationTaskEntry({
    id,
    type: payload.type,
    raw: payload.raw ?? payload,
  })
}

function looksLikeSkillRefinementResult(content: string) {
  try {
    const parsed = JSON.parse(content) as unknown
    return (
      isRecord(parsed) &&
      parsed.type === 'skill_refinement_result' &&
      typeof parsed.status === 'string'
    )
  } catch {
    return false
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return value != null && typeof value === 'object' && !Array.isArray(value)
}
