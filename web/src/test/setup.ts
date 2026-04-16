import { vi } from 'vitest'

if (typeof window !== 'undefined' && typeof window.matchMedia !== 'function') {
  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    value: vi.fn().mockImplementation((query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  })
}

// JSDOM lacks Element.animate, which Svelte transitions call for slide / fly.
// Provide a no-op that resolves immediately so transition-driven components
// render synchronously in tests.
if (typeof window !== 'undefined' && typeof Element.prototype.animate !== 'function') {
  Object.defineProperty(Element.prototype, 'animate', {
    writable: true,
    configurable: true,
    value: function mockAnimate() {
      const finished = Promise.resolve()
      return {
        cancel() {},
        finish() {},
        play() {},
        pause() {},
        reverse() {},
        addEventListener() {},
        removeEventListener() {},
        finished,
        ready: finished,
      }
    },
  })
}
