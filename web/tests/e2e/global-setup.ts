import { resetPerfResults } from './perf'

const baseURL = process.env.PLAYWRIGHT_BASE_URL ?? 'http://127.0.0.1:4173'
const prewarmPaths = [
  '/orgs/org-e2e/projects/project-e2e/settings',
  '/orgs/org-e2e/projects/project-e2e/machines',
]

async function prewarmRouteCache() {
  for (const path of prewarmPaths) {
    const response = await fetch(new URL(path, baseURL), {
      headers: {
        accept: 'text/html',
      },
    })
    if (!response.ok) {
      throw new Error(`Failed to prewarm ${path}: ${response.status} ${response.statusText}`)
    }
  }
}

async function globalSetup() {
  await resetPerfResults()
  await prewarmRouteCache()
}

export default globalSetup
