import type { MachineDetectedArch, MachineDetectedOS, MachineDetectionStatus } from './types'

export function normalizeDetectedOS(value: string | null | undefined): MachineDetectedOS {
  return value === 'darwin' || value === 'linux' ? value : 'unknown'
}

export function normalizeDetectedArch(value: string | null | undefined): MachineDetectedArch {
  return value === 'amd64' || value === 'arm64' ? value : 'unknown'
}

export function normalizeDetectionStatus(value: string | null | undefined): MachineDetectionStatus {
  return value === 'pending' || value === 'ok' || value === 'degraded' || value === 'unknown'
    ? value
    : 'unknown'
}

export function machineDetectedOSLabel(value: string | null | undefined): string {
  switch (normalizeDetectedOS(value)) {
    case 'darwin':
      return 'macOS'
    case 'linux':
      return 'Linux'
    default:
      return 'unknown'
  }
}

export function machineDetectedArchLabel(value: string | null | undefined): string {
  switch (normalizeDetectedArch(value)) {
    case 'amd64':
      return 'amd64'
    case 'arm64':
      return 'arm64'
    default:
      return 'unknown'
  }
}

export function machineDetectionStatusLabel(value: string | null | undefined): string {
  switch (normalizeDetectionStatus(value)) {
    case 'ok':
      return 'Detected'
    case 'degraded':
      return 'Degraded'
    case 'pending':
      return 'Pending'
    default:
      return 'Unknown'
  }
}

export function machineDetectionBadgeClass(value: string | null | undefined): string {
  switch (normalizeDetectionStatus(value)) {
    case 'ok':
      return 'border-emerald-500/30 bg-emerald-500/12 text-emerald-700'
    case 'degraded':
      return 'border-amber-500/30 bg-amber-500/14 text-amber-700'
    case 'pending':
      return 'border-sky-500/30 bg-sky-500/12 text-sky-700'
    default:
      return 'border-slate-500/20 bg-slate-500/10 text-slate-700'
  }
}
