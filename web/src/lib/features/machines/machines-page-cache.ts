import { subscribeOrganizationMachineEvents } from '$lib/features/org-events'
import type { MachineItem, MachineSnapshot } from './types'

type MachinesListSnapshot = {
  machines: MachineItem[]
  selectedId: string
  searchQuery: string
  cachedAt: number
}

type MachineSnapshotState = {
  snapshot: MachineSnapshot | null
  dirty: boolean
}

type MachinesPageCacheRuntime = {
  orgId: string
  listSnapshot: MachinesListSnapshot | null
  listDirty: boolean
  snapshots: Map<string, MachineSnapshotState>
  unsubscribe: (() => void) | null
}

const runtimes = new Map<string, MachinesPageCacheRuntime>()

export function readMachinesPageCache(orgId: string) {
  const runtime = getRuntime(orgId)
  if (!runtime.listSnapshot) {
    return null
  }

  return {
    snapshot: runtime.listSnapshot,
    dirty: runtime.listDirty,
  }
}

export function writeMachinesPageCache(
  orgId: string,
  value: {
    machines: MachineItem[]
    selectedId: string
    searchQuery: string
  },
) {
  const runtime = getRuntime(orgId)
  runtime.listSnapshot = {
    ...value,
    cachedAt: Date.now(),
  }
  runtime.listDirty = false
}

export function markMachinesPageCacheDirty(orgId: string) {
  const runtime = getRuntime(orgId)
  if (!runtime.listSnapshot) {
    return
  }
  runtime.listDirty = true
  for (const snapshotState of runtime.snapshots.values()) {
    snapshotState.dirty = true
  }
}

export function readMachineSnapshotCache(orgId: string, machineId: string) {
  const runtime = getRuntime(orgId)
  const state = runtime.snapshots.get(machineId)
  if (!state) {
    return null
  }

  return {
    snapshot: state.snapshot,
    dirty: state.dirty,
  }
}

export function writeMachineSnapshotCache(
  orgId: string,
  machineId: string,
  snapshot: MachineSnapshot | null,
) {
  const runtime = getRuntime(orgId)
  runtime.snapshots.set(machineId, {
    snapshot,
    dirty: false,
  })
}

export function invalidateMachinesPageCache(orgId?: string) {
  if (!orgId) {
    for (const runtime of runtimes.values()) {
      runtime.listSnapshot = null
      runtime.listDirty = true
      runtime.snapshots.clear()
    }
    return
  }

  const runtime = runtimes.get(orgId)
  if (!runtime) {
    return
  }

  runtime.listSnapshot = null
  runtime.listDirty = true
  runtime.snapshots.clear()
}

export function resetMachinesPageCacheForTests() {
  for (const runtime of runtimes.values()) {
    runtime.unsubscribe?.()
  }
  runtimes.clear()
}

function getRuntime(orgId: string) {
  const existing = runtimes.get(orgId)
  if (existing) {
    return existing
  }

  const created: MachinesPageCacheRuntime = {
    orgId,
    listSnapshot: null,
    listDirty: false,
    snapshots: new Map(),
    unsubscribe: null,
  }
  created.unsubscribe = subscribeOrganizationMachineEvents(orgId, () => {
    if (!created.listSnapshot) {
      return
    }
    created.listDirty = true
    for (const snapshotState of created.snapshots.values()) {
      snapshotState.dirty = true
    }
  })
  runtimes.set(orgId, created)
  return created
}
