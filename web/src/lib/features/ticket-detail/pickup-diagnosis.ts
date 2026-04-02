export type TicketPickupDiagnosis = {
  state: 'runnable' | 'waiting' | 'blocked' | 'running' | 'completed' | 'unavailable'
  primaryReasonCode: string
  primaryReasonMessage: string
  nextActionHint?: string
  reasons: Array<{
    code: string
    message: string
    severity: 'info' | 'warning' | 'error'
  }>
  workflow?: {
    id: string
    name: string
    isActive: boolean
    pickupStatusMatch: boolean
  }
  agent?: {
    id: string
    name: string
    runtimeControlState: string
  }
  provider?: {
    id: string
    name: string
    machineId: string
    machineName: string
    machineStatus: string
    availabilityState: string
    availabilityReason?: string
  }
  retry: {
    attemptCount: number
    retryPaused: boolean
    pauseReason?: string
    nextRetryAt?: string
  }
  capacity: {
    workflow: { limited: boolean; activeRuns: number; capacity: number }
    project: { limited: boolean; activeRuns: number; capacity: number }
    provider: { limited: boolean; activeRuns: number; capacity: number }
    status: { limited: boolean; activeRuns: number; capacity?: number }
  }
  blockedBy: Array<{
    id: string
    identifier: string
    title: string
    statusId: string
    statusName: string
  }>
}
