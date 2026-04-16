export type ProjectReconnectRecovery = {
  sequence: number
}

// Reconnect recovery runs should refetch authoritative state after an outage window.
// This helper coalesces quick retry/live flaps into a single in-order recovery pipeline.
export function createProjectReconnectRecoveryTask(
  recover: (recovery: ProjectReconnectRecovery) => Promise<void> | void,
) {
  let running = false
  let queued: ProjectReconnectRecovery | null = null

  const drain = async () => {
    if (running) {
      return
    }

    running = true
    try {
      while (queued) {
        const nextRecovery = queued
        queued = null
        await recover(nextRecovery)
      }
    } finally {
      running = false
      if (queued) {
        void drain()
      }
    }
  }

  return (recovery: ProjectReconnectRecovery) => {
    queued = recovery
    void drain()
  }
}
