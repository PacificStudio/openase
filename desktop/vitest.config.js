const { defineConfig } = require('vitest/config')

module.exports = defineConfig({
  test: {
    environment: 'node',
    include: ['tests/unit/**/*.test.js', 'tests/integration/**/*.test.js'],
    globals: true,
    restoreMocks: true,
    clearMocks: true,
    mockReset: true,
  },
})
