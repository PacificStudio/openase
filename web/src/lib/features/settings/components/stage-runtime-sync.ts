import { ApiError } from '$lib/api/client'
import type { connectEventStream } from '$lib/api/sse'
type LoadMode = 'initial' | 'background'

type StageRuntimeSyncOptions<TSnapshot> = {
  projectId: string
  loadSnapshot: (projectId: string) => Promise<TSnapshot>
  connectEventStream: typeof connectEventStream
  applySnapshot: (payload: TSnapshot) => void
  skipInitialLoad?: boolean
  setLoading?: (loading: boolean) => void
  onInitialError?: (message: string) => void
  onRefreshError?: (error: unknown) => void
}

export function startStageRuntimeSync<TSnapshot>(options: StageRuntimeSyncOptions<TSnapshot>) {
  let active = true
  let requestVersion = 0
  let queuedReload = false
  let reloadInFlight = false

  const isStaleLoad = (version: number) => !active || version !== requestVersion
  const setInitialLoading = (mode: LoadMode, loading: boolean) => {
    if (mode === 'initial') {
      options.setLoading?.(loading)
    }
  }

  const reportLoadError = (mode: LoadMode, error: unknown) => {
    if (mode === 'initial') {
      options.onInitialError?.(
        error instanceof ApiError ? error.detail : 'Failed to load statuses.',
      )
      return
    }
    options.onRefreshError?.(error)
  }

  async function load(mode: LoadMode) {
    const version = ++requestVersion
    setInitialLoading(mode, true)

    try {
      const payload = await options.loadSnapshot(options.projectId)
      if (isStaleLoad(version)) return
      options.applySnapshot(payload)
    } catch (error) {
      if (isStaleLoad(version)) return
      reportLoadError(mode, error)
    } finally {
      if (!isStaleLoad(version)) {
        setInitialLoading(mode, false)
      }
    }
  }

  async function drainReloadQueue() {
    if (!queuedReload || reloadInFlight || !active) {
      return
    }

    reloadInFlight = true
    queuedReload = false
    try {
      await load('background')
    } finally {
      reloadInFlight = false
      if (queuedReload && active) {
        void drainReloadQueue()
      }
    }
  }

  if (!options.skipInitialLoad) {
    void load('initial')
  }

  const disconnect = options.connectEventStream(
    `/api/v1/projects/${options.projectId}/tickets/stream`,
    {
      onEvent: () => {
        queuedReload = true
        void drainReloadQueue()
      },
      onError: (error) => {
        console.error('Stage runtime ticket stream error:', error)
      },
    },
  )

  return () => {
    active = false
    disconnect()
  }
}
