const { enforceSingleInstanceLock } = require('../../main/runtime/single-instance')

describe('enforceSingleInstanceLock', () => {
  it('quits when the lock cannot be acquired', () => {
    const app = {
      requestSingleInstanceLock: () => false,
      quit: vi.fn(),
      on: vi.fn(),
    }

    expect(enforceSingleInstanceLock(app)).toBe(false)
    expect(app.quit).toHaveBeenCalledTimes(1)
  })

  it('registers a second-instance callback when the lock succeeds', () => {
    const app = {
      requestSingleInstanceLock: () => true,
      quit: vi.fn(),
      on: vi.fn(),
    }
    const callback = vi.fn()

    expect(enforceSingleInstanceLock(app, callback)).toBe(true)
    expect(app.on).toHaveBeenCalledWith('second-instance', callback)
  })
})
