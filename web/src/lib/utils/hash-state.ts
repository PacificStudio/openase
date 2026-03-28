import { browser } from '$app/environment'

export function readHashSelection<T extends string>(
  allowed: readonly T[],
  fallback: T,
  rawHash: string,
): T {
  const value = rawHash.startsWith('#') ? rawHash.slice(1) : rawHash
  return allowed.includes(value as T) ? (value as T) : fallback
}

export function currentHashSelection<T extends string>(allowed: readonly T[], fallback: T): T {
  if (!browser) {
    return fallback
  }

  return readHashSelection(allowed, fallback, window.location.hash)
}

export function writeHashSelection(value: string) {
  if (!browser) {
    return
  }

  const nextHash = `#${value}`
  if (window.location.hash === nextHash) {
    return
  }

  window.location.hash = value
}
