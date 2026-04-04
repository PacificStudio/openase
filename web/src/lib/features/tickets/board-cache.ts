import { subscribeProjectEvents } from '$lib/features/project-events'
import type { BoardData } from '$lib/features/board'

type BoardSnapshot = BoardData & {
  cachedAt: number
}

type BoardCacheRuntime = {
  projectId: string
  snapshot: BoardSnapshot | null
  dirty: boolean
  statusSyncVersion: number | null
  unsubscribe: (() => void) | null
}

const runtimes = new Map<string, BoardCacheRuntime>()

export function readProjectBoardCache(projectId: string) {
  const runtime = getRuntime(projectId)
  if (!runtime.snapshot) {
    return null
  }

  return {
    snapshot: runtime.snapshot,
    dirty: runtime.dirty,
  }
}

export function writeProjectBoardCache(projectId: string, snapshot: BoardData) {
  const runtime = getRuntime(projectId)
  runtime.snapshot = {
    ...snapshot,
    cachedAt: Date.now(),
  }
  runtime.dirty = false
}

export function syncProjectBoardCacheStatusVersion(projectId: string, version: number) {
  const runtime = getRuntime(projectId)
  if (runtime.statusSyncVersion === null) {
    runtime.statusSyncVersion = version
    return
  }

  if (runtime.statusSyncVersion !== version) {
    runtime.statusSyncVersion = version
    runtime.snapshot = null
    runtime.dirty = true
  }
}

export function markProjectBoardCacheDirty(projectId: string) {
  const runtime = getRuntime(projectId)
  if (!runtime.snapshot) {
    return
  }
  runtime.dirty = true
}

export function invalidateProjectBoardCache(projectId?: string) {
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

export function resetProjectBoardCacheForTests() {
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

  const created: BoardCacheRuntime = {
    projectId,
    snapshot: null,
    dirty: false,
    statusSyncVersion: null,
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
