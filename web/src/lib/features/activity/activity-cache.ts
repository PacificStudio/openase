import { subscribeProjectEvents } from '$lib/features/project-events'
import type { ActivityEntry } from './types'

type ActivitySnapshot = {
  entries: ActivityEntry[]
  cachedAt: number
}

type ActivityCacheRuntime = {
  projectId: string
  snapshot: ActivitySnapshot | null
  dirty: boolean
  unsubscribe: (() => void) | null
}

const runtimes = new Map<string, ActivityCacheRuntime>()

export function readProjectActivityCache(projectId: string) {
  const runtime = getRuntime(projectId)
  if (!runtime.snapshot) {
    return null
  }

  return {
    snapshot: runtime.snapshot,
    dirty: runtime.dirty,
  }
}

export function writeProjectActivityCache(projectId: string, entries: ActivityEntry[]) {
  const runtime = getRuntime(projectId)
  runtime.snapshot = {
    entries,
    cachedAt: Date.now(),
  }
  runtime.dirty = false
}

export function markProjectActivityCacheDirty(projectId: string) {
  const runtime = getRuntime(projectId)
  if (!runtime.snapshot) {
    return
  }
  runtime.dirty = true
}

export function invalidateProjectActivityCache(projectId?: string) {
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

export function resetProjectActivityCacheForTests() {
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

  const created: ActivityCacheRuntime = {
    projectId,
    snapshot: null,
    dirty: false,
    unsubscribe: null,
  }
  created.unsubscribe = subscribeProjectEvents(projectId, () => {
    if (created.snapshot) {
      created.dirty = true
    }
  })
  runtimes.set(projectId, created)
  return created
}
