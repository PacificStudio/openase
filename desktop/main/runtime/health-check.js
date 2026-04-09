const { setTimeout: delay } = require('node:timers/promises')

async function waitForHealthy(options) {
  const fetchImpl = options.fetchImpl ?? globalThis.fetch
  const baseURL = options.baseURL.replace(/\/$/, '')
  const timeoutMs = options.timeoutMs ?? 45_000
  const intervalMs = options.intervalMs ?? 250
  const startedAt = Date.now()
  const urls = [`${baseURL}/healthz`, `${baseURL}/api/v1/healthz`]
  let lastError = null

  if (typeof fetchImpl !== 'function') {
    throw new Error('fetch implementation is required for desktop health checks')
  }

  while (Date.now() - startedAt <= timeoutMs) {
    try {
      for (const url of urls) {
        const response = await fetchImpl(url, {
          headers: { accept: 'application/json' },
        })
        if (!response.ok) {
          throw new Error(`health check ${url} returned HTTP ${response.status}`)
        }
      }

      return {
        baseURL,
        checkedAt: new Date().toISOString(),
      }
    } catch (error) {
      lastError = error
      await delay(intervalMs)
    }
  }

  throw new Error(`timed out after ${timeoutMs}ms waiting for ${baseURL}: ${lastError ? lastError.message : 'unknown error'}`)
}

module.exports = {
  waitForHealthy,
}
