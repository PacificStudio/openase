import { isProjectUpdateEvent, subscribeProjectEvents } from '$lib/features/project-events'
import type { ProjectUpdateThread } from './types'

type ProjectUpdatesSnapshot = {
  threads: ProjectUpdateThread[]
  nextCursor: string
  hasMoreThreads: boolean
  loadedMorePages: boolean
  cachedAt: number
}

type ProjectUpdatesCacheRuntime = {
  projectId: string
  snapshot: ProjectUpdatesSnapshot | null
  dirty: boolean
  unsubscribe: (() => void) | null
}

const runtimes = new Map<string, ProjectUpdatesCacheRuntime>()

export function readProjectUpdatesCache(projectId: string) {
  const runtime = getRuntime(projectId)
  if (!runtime.snapshot) {
    return null
  }

  return {
    snapshot: runtime.snapshot,
    dirty: runtime.dirty,
  }
}

export function writeProjectUpdatesCache(
  projectId: string,
  snapshot: Omit<ProjectUpdatesSnapshot, 'cachedAt'>,
) {
  const runtime = getRuntime(projectId)
  runtime.snapshot = {
    ...snapshot,
    cachedAt: Date.now(),
  }
  runtime.dirty = false
}

export function markProjectUpdatesCacheDirty(projectId: string) {
  const runtime = getRuntime(projectId)
  if (!runtime.snapshot) {
    return
  }
  runtime.dirty = true
}

export function invalidateProjectUpdatesCache(projectId?: string) {
  if (!projectId) {
    for (const runtime of runtimes.values()) {
      runtime.snapshot = null
      runtime.dirty = true
    }
    return
  }

  const runtime = runtimes.get(projectId)
  if (!runtime) {
    return
  }

  runtime.snapshot = null
  runtime.dirty = true
}

export function resetProjectUpdatesCacheForTests() {
  for (const runtime of runtimes.values()) {
    runtime.unsubscribe?.()
  }
  runtimes.clear()
}

function getRuntime(projectId: string) {
  const existing = runtimes.get(projectId)
  if (existing) {
    return existing
  }

  const created: ProjectUpdatesCacheRuntime = {
    projectId,
    snapshot: null,
    dirty: false,
    unsubscribe: null,
  }
  created.unsubscribe = subscribeProjectEvents(projectId, (event) => {
    if (created.snapshot && isProjectUpdateEvent(event)) {
      created.dirty = true
    }
  })
  runtimes.set(projectId, created)
  return created
}
