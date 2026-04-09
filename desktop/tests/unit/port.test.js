const { findAvailablePort } = require('../../main/runtime/port')

describe('findAvailablePort', () => {
  it('returns a usable ephemeral localhost port', async () => {
    const port = await findAvailablePort('127.0.0.1')

    expect(port).toBeGreaterThan(0)
    expect(port).toBeLessThan(65_536)
  })
})
