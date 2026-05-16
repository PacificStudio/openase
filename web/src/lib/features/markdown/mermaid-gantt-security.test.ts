import { execFile } from 'node:child_process'
import path from 'node:path'
import { promisify } from 'node:util'
import { describe, expect, it } from 'vitest'

const execFileAsync = promisify(execFile)

describe('mermaid gantt security regression', () => {
  it('renders hostile excludes input without hanging', async () => {
    const scriptPath = path.resolve(process.cwd(), 'scripts/verify-mermaid-gantt-no-hang.mjs')
    const { stdout } = await execFileAsync(process.execPath, [scriptPath], {
      cwd: process.cwd(),
      env: {
        ...process.env,
        MERMAID_RENDER_TIMEOUT_MS: '1500',
      },
      timeout: 8000,
    })

    const result = JSON.parse(stdout.trim()) as {
      mermaidPath: string
      timeoutMs: number
      status: 'rendered' | 'rejected'
      svgLength?: number
      errorMessage?: string
    }

    expect(result.mermaidPath).toContain(`${path.sep}streamdown-svelte@`)
    expect(result.timeoutMs).toBe(1500)
    expect(['rendered', 'rejected']).toContain(result.status)
    if (result.status === 'rendered') {
      expect(result.svgLength).toBeGreaterThan(0)
      return
    }

    expect(result.errorMessage).toContain('excludes')
  }, 5_000)
})
