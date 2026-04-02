import { readdirSync, readFileSync, statSync } from 'node:fs'
import path from 'node:path'
import { describe, expect, it } from 'vitest'

const libRoot = path.resolve(import.meta.dirname, '../..')

const allowedConnectEventStreamImports = new Set([
  path.join(libRoot, 'api/sse.ts'),
  path.join(libRoot, 'features/org-events/org-event-bus.ts'),
  path.join(libRoot, 'features/project-events/project-event-bus.ts'),
  path.join(libRoot, 'features/agents/components/agents-page-streams.ts'),
])

const allowedProjectStreamSources = new Set([
  path.join(libRoot, 'features/project-events/project-event-bus.ts'),
  path.join(libRoot, 'testing/e2e/mock-api.ts'),
])

describe('project passive event bus guard', () => {
  it('keeps connectEventStream imports behind approved boundaries', () => {
    const violations = listSourceFiles(libRoot)
      .filter((file) => !file.endsWith('.test.ts'))
      .filter((file) => !file.includes(`${path.sep}api${path.sep}generated${path.sep}`))
      .filter((file) => !allowedConnectEventStreamImports.has(file))
      .filter((file) => readFileSync(file, 'utf8').includes("from '$lib/api/sse'"))
      .filter((file) => readFileSync(file, 'utf8').includes('connectEventStream'))

    expect(violations).toEqual([])
  })

  it('forbids direct project-scoped passive stream URLs outside the bus runtime', () => {
    const violations = listSourceFiles(libRoot)
      .filter((file) => !file.endsWith('.test.ts'))
      .filter((file) => !file.includes(`${path.sep}api${path.sep}generated${path.sep}`))
      .filter((file) => !allowedProjectStreamSources.has(file))
      .filter((file) => {
        const source = readFileSync(file, 'utf8')
        return (
          source.includes('/api/v1/projects/${') &&
          source.includes('/stream') &&
          !source.includes('/agents/${agentId}/output/stream') &&
          !source.includes('/agents/${agentId}/steps/stream')
        )
      })

    expect(violations).toEqual([])
  })
})

function listSourceFiles(root: string): string[] {
  const entries = readdirSync(root)
  const files: string[] = []

  for (const entry of entries) {
    const absolute = path.join(root, entry)
    const stats = statSync(absolute)
    if (stats.isDirectory()) {
      files.push(...listSourceFiles(absolute))
      continue
    }
    if (absolute.endsWith('.ts') || absolute.endsWith('.svelte')) {
      files.push(absolute)
    }
  }

  return files
}
