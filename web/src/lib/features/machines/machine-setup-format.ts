import { i18nStore } from '$lib/i18n/store.svelte'

export function buildSSHHelperPreview(input: {
  host: string | null | undefined
  sshUser: string | null | undefined
  sshKeyPath: string | null | undefined
}): string | null {
  const host = (input.host ?? '').trim()
  const sshUser = (input.sshUser ?? '').trim()
  if (!host || !sshUser || host.toLowerCase() === 'local') {
    return null
  }

  const sshKeyPath = (input.sshKeyPath ?? '').trim()
  const keyArg = sshKeyPath ? ` -i ${sshKeyPath}` : ''
  return `ssh${keyArg} ${sshUser}@${host}`
}

export function friendlyTransportLabel(value: string | null | undefined): string {
  switch ((value ?? '').trim()) {
    case 'local':
      return i18nStore.t('machines.transport.local')
    case 'ssh':
      return i18nStore.t('machines.transport.ssh')
    case 'ws_listener':
      return i18nStore.t('machines.transport.wsListener')
    case 'ws_reverse':
      return i18nStore.t('machines.transport.wsReverse')
    default:
      return value?.trim() ? value.trim() : i18nStore.t('machines.transport.unknown')
  }
}

export function normalizeSessionState(value: string | null | undefined): string {
  const raw = (value ?? '').trim()
  if (!raw) return 'unknown'
  return raw
}

export function humanizeSessionState(value: string): string {
  switch (value) {
    case 'connected':
      return i18nStore.t('machines.sessionState.connected')
    case 'disconnected':
      return i18nStore.t('machines.sessionState.disconnected')
    case 'unavailable':
      return i18nStore.t('machines.sessionState.unavailable')
    default:
      return i18nStore.t('machines.sessionState.unknown')
  }
}
