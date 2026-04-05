import { cleanup, fireEvent, render } from '@testing-library/svelte'
import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest'

const { toastError } = vi.hoisted(() => ({
  toastError: vi.fn(),
}))

vi.mock('$lib/stores/toast.svelte', () => ({
  toastStore: {
    success: vi.fn(),
    error: toastError,
  },
}))

vi.mock('$lib/stores/app.svelte', () => ({
  appStore: {
    currentProject: { default_agent_provider_id: 'provider-1' },
  },
}))

import type { AgentProvider } from '$lib/api/contracts'
import HarnessAiSidebar from './harness-ai-sidebar.svelte'

const encoder = new TextEncoder()

const providerFixtures: AgentProvider[] = [
  {
    id: 'provider-1',
    organization_id: 'org-1',
    machine_id: 'machine-1',
    machine_name: 'Localhost',
    machine_host: '127.0.0.1',
    machine_status: 'online',
    machine_ssh_user: null,
    machine_workspace_root: '/workspace',
    name: 'Codex',
    adapter_type: 'codex-app-server',
    permission_profile: 'unrestricted',
    availability_state: 'available',
    available: true,
    availability_checked_at: '2026-03-28T12:00:00Z',
    availability_reason: null,
    capabilities: {
      ephemeral_chat: {
        state: 'available',
        reason: null,
      },
      harness_ai: {
        state: 'available',
        reason: null,
      },
      skill_ai: {
        state: 'available',
        reason: null,
      },
    },
    cli_command: 'codex',
    cli_args: [],
    auth_config: {},
    cli_rate_limit: null,
    cli_rate_limit_updated_at: null,
    model_name: 'gpt-5.4',
    model_temperature: 0,
    model_max_tokens: 4096,
    max_parallel_runs: 2,
    cost_per_input_token: 0,
    cost_per_output_token: 0,
    pricing_config: {},
  },
]

const harnessContent = [
  '---',
  'name: Coding Workflow',
  'type: coding',
  '---',
  '',
  'Write clean, tested code.',
].join('\n')

type ChunkedTurnPlan = {
  expectedSessionId?: string
  sessionId: string
  turnsUsed: number
  turnsRemaining: number
  assistantChunks: string[]
}

type TimedSSEEvent = {
  atMs: number
  event: string
  payload: Record<string, unknown>
}

type RecordedRequest = {
  method: string
  url: string
  headers: Headers
  body?: Record<string, unknown>
}

function formatSSEFrame(event: string, payload: Record<string, unknown>) {
  return `event: ${event}\ndata: ${JSON.stringify(payload)}\n\n`
}

function splitIntoUnevenChunks(frame: string) {
  if (frame.length < 3) {
    return [frame]
  }

  const firstBreak = Math.max(1, Math.floor(frame.length / 3))
  const secondBreak = Math.max(firstBreak + 1, Math.floor((frame.length * 2) / 3))
  return [
    frame.slice(0, firstBreak),
    frame.slice(firstBreak, secondBreak),
    frame.slice(secondBreak),
  ]
}

function buildTimedChunks(plan: ChunkedTurnPlan) {
  const events: TimedSSEEvent[] = [
    {
      atMs: 0,
      event: 'session',
      payload: { session_id: plan.sessionId },
    },
    {
      atMs: 15_000,
      event: 'message',
      payload: { type: 'text', content: plan.assistantChunks[0] },
    },
    {
      atMs: 30_000,
      event: 'message',
      payload: { type: 'text', content: plan.assistantChunks[1] },
    },
    {
      atMs: 45_000,
      event: 'message',
      payload: { type: 'text', content: plan.assistantChunks[2] },
    },
    {
      atMs: 60_000,
      event: 'done',
      payload: {
        session_id: plan.sessionId,
        turns_used: plan.turnsUsed,
        turns_remaining: plan.turnsRemaining,
      },
    },
  ]

  return buildTimedChunksFromEvents(events)
}

function buildTimedChunksFromEvents(events: TimedSSEEvent[]) {
  return events.flatMap(({ atMs, event, payload }) =>
    splitIntoUnevenChunks(formatSSEFrame(event, payload)).map((chunk, index) => ({
      atMs: atMs + index,
      chunk,
    })),
  )
}

function createChunkedChatStream(plan: ChunkedTurnPlan, signal?: AbortSignal) {
  const scheduled: number[] = []

  return new ReadableStream<Uint8Array>({
    start(controller) {
      let closed = false

      const closeSafely = () => {
        if (closed) {
          return
        }
        closed = true
        controller.close()
      }

      const abortStream = () => {
        for (const handle of scheduled) {
          window.clearTimeout(handle)
        }
        if (closed) {
          return
        }
        closed = true
        controller.error(new DOMException('Aborted', 'AbortError'))
      }

      signal?.addEventListener('abort', abortStream, { once: true })

      const timedChunks = buildTimedChunks(plan)
      for (const [index, chunk] of timedChunks.entries()) {
        const handle = window.setTimeout(() => {
          if (closed) {
            return
          }
          controller.enqueue(encoder.encode(chunk.chunk))
          if (index === timedChunks.length - 1) {
            closeSafely()
          }
        }, chunk.atMs)
        scheduled.push(handle)
      }
    },
  })
}

function createTimedEventStream(events: TimedSSEEvent[], signal?: AbortSignal) {
  const scheduled: number[] = []

  return new ReadableStream<Uint8Array>({
    start(controller) {
      let closed = false

      const closeSafely = () => {
        if (closed) {
          return
        }
        closed = true
        controller.close()
      }

      const abortStream = () => {
        for (const handle of scheduled) {
          window.clearTimeout(handle)
        }
        if (closed) {
          return
        }
        closed = true
        controller.error(new DOMException('Aborted', 'AbortError'))
      }

      signal?.addEventListener('abort', abortStream, { once: true })

      const timedChunks = buildTimedChunksFromEvents(events)
      for (const [index, chunk] of timedChunks.entries()) {
        const handle = window.setTimeout(() => {
          if (closed) {
            return
          }
          controller.enqueue(encoder.encode(chunk.chunk))
          if (index === timedChunks.length - 1) {
            closeSafely()
          }
        }, chunk.atMs)
        scheduled.push(handle)
      }
    },
  })
}

async function advanceLongTurn() {
  for (let step = 0; step <= 60; step += 1) {
    await vi.advanceTimersByTimeAsync(1_000)
  }
}

async function flushDom() {
  for (let index = 0; index < 12; index += 1) {
    await Promise.resolve()
  }
}

describe('HarnessAiSidebar long streaming', () => {
  beforeAll(() => {
    HTMLElement.prototype.scrollIntoView ??= vi.fn()
    HTMLElement.prototype.hasPointerCapture ??= vi.fn(() => false)
    HTMLElement.prototype.releasePointerCapture ??= vi.fn()
    globalThis.ResizeObserver ??= class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  })

  afterEach(() => {
    cleanup()
    vi.clearAllMocks()
    vi.useRealTimers()
    vi.unstubAllGlobals()
    window.localStorage.clear()
  })

  it('finishes two 120-second chunked chat streams without interruption and keeps the transcript correct', async () => {
    vi.useFakeTimers()

    const turnPlans: ChunkedTurnPlan[] = [
      {
        sessionId: 'session-harness-long-1',
        turnsUsed: 1,
        turnsRemaining: 9,
        assistantChunks: [
          'I reviewed the current harness and found two weak spots. ',
          'First, the coding constraints need explicit test expectations. ',
          'Second, the rollback guidance should be tighter before I propose an edit.',
        ],
      },
      {
        expectedSessionId: 'session-harness-long-1',
        sessionId: 'session-harness-long-1',
        turnsUsed: 2,
        turnsRemaining: 8,
        assistantChunks: [
          'Proposed update: keep parse-not-validate at the boundary. ',
          'Add a rule that every workflow change must ship with a targeted regression test. ',
          'I can convert that into a harness patch next if you want.',
        ],
      },
    ]

    const recordedRequests: RecordedRequest[] = []
    const abortedTurns: Array<{ get value(): boolean }> = []
    let turnIndex = 0

    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const url =
          typeof input === 'string' ? input : input instanceof URL ? input.toString() : input.url
        const method = init?.method ?? 'GET'
        const headers = new Headers(init?.headers)

        if (method === 'POST' && url === '/api/v1/chat') {
          const body = init?.body
            ? (JSON.parse(String(init.body)) as Record<string, unknown>)
            : undefined
          recordedRequests.push({ method, url, headers, body })

          const plan = turnPlans[turnIndex]
          turnIndex += 1
          expect(body?.session_id).toBe(plan.expectedSessionId)

          let aborted = false
          init?.signal?.addEventListener(
            'abort',
            () => {
              aborted = true
            },
            { once: true },
          )
          abortedTurns.push({
            get value() {
              return aborted
            },
          })

          return new Response(createChunkedChatStream(plan, init?.signal ?? undefined), {
            status: 200,
            headers: { 'Content-Type': 'text/event-stream' },
          })
        }

        if (method === 'DELETE' && url.startsWith('/api/v1/chat/')) {
          recordedRequests.push({ method, url, headers })
          return new Response(null, { status: 204 })
        }

        throw new Error(`Unexpected fetch request: ${method} ${url}`)
      }),
    )

    const { container, getByPlaceholderText, queryByText } = render(HarnessAiSidebar, {
      props: {
        projectId: 'project-1',
        workflowId: 'workflow-1',
        providers: providerFixtures,
        draftContent: harnessContent,
      },
    })

    const prompt = getByPlaceholderText('Ask AI to refine this harness…')

    await fireEvent.input(prompt, {
      target: { value: 'Review this harness and tell me what is weak.' },
    })
    await fireEvent.keyDown(prompt, { key: 'Enter' })
    await flushDom()
    await advanceLongTurn()
    await flushDom()

    expect(container.textContent).toContain(
      'I reviewed the current harness and found two weak spots. First, the coding constraints need explicit test expectations. Second, the rollback guidance should be tighter before I propose an edit.',
    )
    expect(queryByText('Thinking…')).toBeNull()

    await fireEvent.input(prompt, {
      target: { value: 'Give me the exact update guidance next.' },
    })
    await fireEvent.keyDown(prompt, { key: 'Enter' })
    await flushDom()
    await advanceLongTurn()
    await flushDom()

    expect(container.textContent).toContain(
      'Proposed update: keep parse-not-validate at the boundary. Add a rule that every workflow change must ship with a targeted regression test. I can convert that into a harness patch next if you want.',
    )
    expect(queryByText('Thinking…')).toBeNull()

    const postRequests = recordedRequests.filter((request) => request.method === 'POST')
    expect(postRequests).toHaveLength(2)
    expect(postRequests[0]?.body).toMatchObject({
      message: 'Review this harness and tell me what is weak.',
      source: 'harness_editor',
      provider_id: 'provider-1',
      context: {
        project_id: 'project-1',
        workflow_id: 'workflow-1',
        harness_draft: harnessContent,
      },
    })
    expect(postRequests[0]?.body?.session_id).toBeUndefined()
    expect(postRequests[0]?.headers.get('accept')).toBe('text/event-stream')
    expect(postRequests[0]?.headers.get('X-OpenASE-Chat-User')).toBeNull()

    expect(postRequests[1]?.body).toMatchObject({
      message: 'Give me the exact update guidance next.',
      source: 'harness_editor',
      provider_id: 'provider-1',
      session_id: 'session-harness-long-1',
      context: {
        project_id: 'project-1',
        workflow_id: 'workflow-1',
        harness_draft: harnessContent,
      },
    })
    expect(turnIndex).toBe(2)
    expect(abortedTurns.map((turn) => turn.value)).toEqual([false, false])
    expect(toastError).not.toHaveBeenCalled()
  })

  it('renders a structured diff card when the stream sends prose followed by a diff event', async () => {
    vi.useFakeTimers()

    const streamEvents: TimedSSEEvent[] = [
      {
        atMs: 0,
        event: 'session',
        payload: { session_id: 'session-harness-regression-1' },
      },
      {
        atMs: 10,
        event: 'message',
        payload: {
          type: 'text',
          content:
            'I will first map the editable areas from the current harness and project status topology, then provide a directly applicable structured diff.',
        },
      },
      {
        atMs: 20,
        event: 'message',
        payload: {
          type: 'diff',
          file: 'harness content',
          hunks: [
            {
              old_start: 6,
              old_lines: 1,
              new_start: 6,
              new_lines: 1,
              lines: [
                { op: 'remove', text: 'Write clean, tested code.' },
                { op: 'add', text: 'Write clean, tested code and require a PR URL before Done.' },
              ],
            },
          ],
        },
      },
      {
        atMs: 30,
        event: 'done',
        payload: {
          session_id: 'session-harness-regression-1',
          turns_used: 1,
          turns_remaining: 9,
        },
      },
    ]

    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const url =
          typeof input === 'string' ? input : input instanceof URL ? input.toString() : input.url
        const method = init?.method ?? 'GET'
        if (method === 'POST' && url === '/api/v1/chat') {
          return new Response(createTimedEventStream(streamEvents, init?.signal ?? undefined), {
            status: 200,
            headers: { 'Content-Type': 'text/event-stream' },
          })
        }

        if (method === 'DELETE' && url.startsWith('/api/v1/chat/')) {
          return new Response(null, { status: 204 })
        }

        throw new Error(`Unexpected fetch request: ${method} ${url}`)
      }),
    )

    const { container, getByPlaceholderText, findByText } = render(HarnessAiSidebar, {
      props: {
        projectId: 'project-1',
        workflowId: 'workflow-1',
        providers: providerFixtures,
        draftContent: harnessContent,
      },
    })

    const prompt = getByPlaceholderText('Ask AI to refine this harness…')
    await fireEvent.input(prompt, {
      target: { value: 'Require this product manager to commit, push, and open a PR before Done.' },
    })
    await fireEvent.keyDown(prompt, { key: 'Enter' })

    await vi.advanceTimersByTimeAsync(100)
    await flushDom()

    expect(
      await findByText(
        'I will first map the editable areas from the current harness and project status topology, then provide a directly applicable structured diff.',
      ),
    ).toBeTruthy()
    expect(await findByText('Structured Diff')).toBeTruthy()
    expect(await findByText('Apply')).toBeTruthy()
    expect(container.textContent).not.toContain('{"type":"diff"')
    expect(container.textContent).not.toContain('"hunks"')
    expect(toastError).not.toHaveBeenCalled()
  })
})
