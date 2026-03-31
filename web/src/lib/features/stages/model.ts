import type { TicketStage } from '$lib/api/contracts'

export type StageDraft = {
  key: string
  name: string
  maxActiveRuns: string
}

export type ParsedStageDraft = {
  key: string
  name: string
  maxActiveRuns: number | null
}

export type EditableStage = ParsedStageDraft & {
  id: string
  position: number
  activeRuns: number
}

type ParseResult<T> = { ok: true; value: T } | { ok: false; error: string }

const STAGE_KEY_SEGMENT_PATTERN = /[a-z0-9]+/g

export function createEmptyStageDraft(): StageDraft {
  return {
    key: '',
    name: '',
    maxActiveRuns: '',
  }
}

export function stageKeyFromName(raw: string) {
  return raw.trim().toLowerCase().match(STAGE_KEY_SEGMENT_PATTERN)?.join('-') ?? ''
}

export function normalizeStages(stages: TicketStage[]): EditableStage[] {
  return stages
    .slice()
    .sort((left, right) => left.position - right.position)
    .map((stage) => ({
      id: stage.id,
      key: stage.key,
      name: stage.name,
      position: stage.position,
      activeRuns: stage.active_runs,
      maxActiveRuns: typeof stage.max_active_runs === 'number' ? stage.max_active_runs : null,
    }))
}

export function parseStageDraft(raw: StageDraft): ParseResult<ParsedStageDraft> {
  const name = raw.name.trim()
  if (!name) {
    return { ok: false, error: 'Stage name is required.' }
  }

  const key = stageKeyFromName(raw.key)
  if (!key) {
    return { ok: false, error: 'Stage key is required.' }
  }

  const maxActiveRuns = raw.maxActiveRuns.trim()
  if (!maxActiveRuns) {
    return {
      ok: true,
      value: {
        key,
        name,
        maxActiveRuns: null,
      },
    }
  }

  const parsed = Number(maxActiveRuns)
  if (!Number.isInteger(parsed) || parsed <= 0) {
    return { ok: false, error: 'Max active runs must be a whole number greater than 0.' }
  }

  return {
    ok: true,
    value: {
      key,
      name,
      maxActiveRuns: parsed,
    },
  }
}

export function moveStage(
  stages: EditableStage[],
  stageId: string,
  direction: 'up' | 'down',
): EditableStage[] {
  const currentIndex = stages.findIndex((stage) => stage.id === stageId)
  if (currentIndex < 0) {
    return stages
  }

  const targetIndex = direction === 'up' ? currentIndex - 1 : currentIndex + 1
  if (targetIndex < 0 || targetIndex >= stages.length) {
    return stages
  }

  const nextStages = stages.slice()
  const [moved] = nextStages.splice(currentIndex, 1)
  nextStages.splice(targetIndex, 0, moved)

  return nextStages.map((stage, index) => ({
    ...stage,
    position: index,
  }))
}
