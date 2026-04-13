import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

import { describe, expect, it } from 'vitest'

import {
  applyTicketRunStreamFrame,
  createEmptyTicketRunTranscriptState,
  hydrateTicketRunDetail,
  setTicketRunList,
} from './run-transcript'
import { mapTicketRunDetail } from './run-transcript-data'
import type { TicketRunDetailPayload } from '$lib/api/contracts'

type ReplayFixture = {
  provider_name: string
  detail: TicketRunDetailPayload
  all_frames: Array<{ event: string; payload: Record<string, unknown> }>
  supplement_frames: Array<{ event: string; payload: Record<string, unknown> }>
}

const legacyClaudeExecutionFailure = 'Claude Code reported an empty error_during_execution result.'
const humanFriendlyClaudeExecutionFailure =
  'Claude Code failed while executing the task. Try again or check the logs for more details.'

function loadFixture(name: string): ReplayFixture {
  const fixturePath = resolve(process.cwd(), 'src/lib/features/ticket-detail/testdata', name)
  return JSON.parse(readFileSync(fixturePath, 'utf-8')) as ReplayFixture
}

function withFriendlyClaudeExecutionFailure(fixture: ReplayFixture): ReplayFixture {
  return JSON.parse(
    JSON.stringify(fixture).replaceAll(
      legacyClaudeExecutionFailure,
      humanFriendlyClaudeExecutionFailure,
    ),
  ) as ReplayFixture
}

function replayFrames(
  frames: ReplayFixture['all_frames'],
  detailPayload: TicketRunDetailPayload,
  hydrateFirst: boolean,
) {
  const detail = mapTicketRunDetail(detailPayload)
  let state = setTicketRunList(createEmptyTicketRunTranscriptState(), [detail.run])
  if (hydrateFirst) {
    state = hydrateTicketRunDetail(state, detail)
  }
  for (const frame of frames) {
    state = applyTicketRunStreamFrame(state, {
      event: frame.event,
      data: JSON.stringify(frame.payload),
    })
  }
  return state
}

function sliceTranscriptPage(detail: ReturnType<typeof mapTicketRunDetail>, itemCount: number) {
  const startIndex = Math.max(detail.transcriptPage.items.length - itemCount, 0)
  const items = detail.transcriptPage.items.slice(startIndex)
  return {
    ...detail,
    transcriptPage: {
      ...detail.transcriptPage,
      items,
      hasOlder: startIndex > 0,
      hiddenOlderCount: startIndex,
      oldestCursor: items[0]?.cursor,
      newestCursor: items.at(-1)?.cursor,
    },
  }
}

describe('ticket run transcript real sample replay', () => {
  it('reaches the same final transcript state when a large Claude run is hydrated then supplemented via stream', () => {
    const fixture = loadFixture('claude-code-replay-fixture.json')
    const detail = mapTicketRunDetail(fixture.detail)

    const streamThenHydrate = hydrateTicketRunDetail(
      replayFrames(fixture.all_frames, fixture.detail, false),
      detail,
    )
    const hydrateThenSupplement = replayFrames(fixture.supplement_frames, fixture.detail, true)

    expect(hydrateThenSupplement).toEqual(streamThenHydrate)
  })

  it('keeps the terminal Codex run snapshot stable when older lifecycle frames arrive after detail hydration', () => {
    const fixture = loadFixture('openai-codex-replay-fixture.json')
    const detail = mapTicketRunDetail(fixture.detail)
    const state = replayFrames(fixture.supplement_frames, fixture.detail, true)

    expect(state.currentRun).toEqual(detail.run)
    expect(state.currentRun?.status).toBe('ended')
  })

  it('reaches the same transcript state when hydrating only the latest page, expanding older history, and then replaying stream supplements', () => {
    const fixture = loadFixture('claude-code-replay-fixture.json')
    const fullDetail = mapTicketRunDetail(fixture.detail)
    const latestPageOnly = sliceTranscriptPage(fullDetail, 120)

    let pagedState = setTicketRunList(createEmptyTicketRunTranscriptState(), [fullDetail.run])
    pagedState = hydrateTicketRunDetail(pagedState, latestPageOnly)
    pagedState = hydrateTicketRunDetail(pagedState, fullDetail, { select: false })
    for (const frame of fixture.supplement_frames) {
      pagedState = applyTicketRunStreamFrame(pagedState, {
        event: frame.event,
        data: JSON.stringify(frame.payload),
      })
    }

    const fullHydrationThenSupplement = replayFrames(
      fixture.supplement_frames,
      fixture.detail,
      true,
    )
    expect(pagedState).toEqual(fullHydrationThenSupplement)
  })

  it('preserves user-facing Claude execution failure text through hydrate and replay', () => {
    const fixture = withFriendlyClaudeExecutionFailure(
      loadFixture('claude-code-replay-fixture.json'),
    )
    const state = replayFrames(fixture.supplement_frames, fixture.detail, true)

    const errorBlock = state.blocks.find(
      (block) =>
        block.kind === 'task_status' &&
        block.statusType === 'error' &&
        block.raw?.subtype === 'error_during_execution',
    )

    expect(errorBlock).toMatchObject({
      detail: humanFriendlyClaudeExecutionFailure,
      raw: { subtype: 'error_during_execution' },
    })
    expect(JSON.stringify(errorBlock)).not.toContain(legacyClaudeExecutionFailure)
  })
})
