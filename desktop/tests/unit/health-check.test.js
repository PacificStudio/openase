const { waitForHealthy } = require('../../main/runtime/health-check')

describe('waitForHealthy', () => {
  it('requires both readiness endpoints to succeed', async () => {
    const fetchImpl = vi.fn(async () => ({ ok: true, status: 200 }))

    const result = await waitForHealthy({
      baseURL: 'http://127.0.0.1:9999',
      timeoutMs: 100,
      intervalMs: 5,
      fetchImpl,
    })

    expect(result.baseURL).toBe('http://127.0.0.1:9999')
    expect(fetchImpl).toHaveBeenCalledTimes(2)
  })

  it('times out with the last error message', async () => {
    await expect(
      waitForHealthy({
        baseURL: 'http://127.0.0.1:9999',
        timeoutMs: 30,
        intervalMs: 5,
        fetchImpl: async () => ({ ok: false, status: 503 }),
      }),
    ).rejects.toThrow(/timed out/i)
  })
})
