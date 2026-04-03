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

function loadFixture(name: string): ReplayFixture {
  const fixturePath = resolve(process.cwd(), 'src/lib/features/ticket-detail/testdata', name)
  return JSON.parse(readFileSync(fixturePath, 'utf-8')) as ReplayFixture
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
})
