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
      return 'Local runtime'
    case 'ssh':
      return 'SSH helper path'
    case 'ws_listener':
      return 'Direct-connect listener'
    case 'ws_reverse':
      return 'Reverse-connect daemon'
    default:
      return value?.trim() ? value.trim() : 'Unknown transport'
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
      return 'Connected'
    case 'disconnected':
      return 'Disconnected'
    case 'unavailable':
      return 'Unavailable'
    default:
      return 'Unknown'
  }
}
