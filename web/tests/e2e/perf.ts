import { expect, type Locator, type Page, type TestInfo } from '@playwright/test'
import { mkdir, readFile, writeFile } from 'node:fs/promises'
import path from 'node:path'

export type PerfMeasurement = {
  scenario: string
  metric: string
  durationMs: number
  budgetMs: number
}

// Keep perf artifacts outside the Vite project root so result writes do not trigger HMR reloads
// mid-test and collapse drawers/dialogs that we are measuring.
const perfResultsPath = path.resolve(
  process.cwd(),
  '../.playwright-artifacts/web-perf-results.json',
)
const enforceBudgets = process.env.PLAYWRIGHT_ENFORCE_BUDGETS === '1'

export async function resetPerfResults() {
  await mkdir(path.dirname(perfResultsPath), { recursive: true })
  await writeFile(perfResultsPath, '[]\n', 'utf8')
}

export async function readPerfResults(): Promise<PerfMeasurement[]> {
  const raw = await readFile(perfResultsPath, 'utf8')
  return JSON.parse(raw) as PerfMeasurement[]
}

export async function measureNavigation(input: {
  page: Page
  ready: Locator
  scenario: string
  budgetMs: number
  action: () => Promise<unknown>
  testInfo: TestInfo
}) {
  return measureAction({
    ...input,
    metric: 'route_to_interactive_ms',
  })
}

export async function measureFeedback(input: {
  ready: Locator
  scenario: string
  budgetMs: number
  action: () => Promise<unknown>
  testInfo: TestInfo
}) {
  return measureAction({
    ...input,
    metric: 'action_feedback_ms',
  })
}

export async function measureCompletion(input: {
  ready: Locator
  scenario: string
  budgetMs: number
  action: () => Promise<unknown>
  testInfo: TestInfo
}) {
  return measureAction({
    ...input,
    metric: 'action_complete_ms',
  })
}

async function measureAction(input: {
  ready: Locator
  scenario: string
  metric: string
  budgetMs: number
  action: () => Promise<unknown>
  testInfo: TestInfo
}) {
  const start = performance.now()
  await input.action()
  await expect(input.ready).toBeVisible()
  const durationMs = Number((performance.now() - start).toFixed(1))

  const record: PerfMeasurement = {
    scenario: input.scenario,
    metric: input.metric,
    durationMs,
    budgetMs: input.budgetMs,
  }

  await appendPerfResult(record)
  await input.testInfo.attach(`${input.scenario}-${input.metric}`, {
    body: JSON.stringify(record, null, 2),
    contentType: 'application/json',
  })

  if (enforceBudgets) {
    expect(
      durationMs,
      `${input.scenario} ${input.metric} exceeded budget ${input.budgetMs}ms`,
    ).toBeLessThan(input.budgetMs)
  }

  return durationMs
}

async function appendPerfResult(record: PerfMeasurement) {
  await mkdir(path.dirname(perfResultsPath), { recursive: true })
  const current = await readPerfResults().catch((): PerfMeasurement[] => [])
  current.push(record)
  await writeFile(perfResultsPath, `${JSON.stringify(current, null, 2)}\n`, 'utf8')
}
