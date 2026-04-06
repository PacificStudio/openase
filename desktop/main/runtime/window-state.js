const fs = require('node:fs')

const defaultWindowState = {
  width: 1440,
  height: 920,
}

function loadWindowState(statePath) {
  try {
    const raw = fs.readFileSync(statePath, 'utf8')
    const parsed = JSON.parse(raw)
    if (
      typeof parsed?.width === 'number' &&
      typeof parsed?.height === 'number' &&
      parsed.width > 0 &&
      parsed.height > 0
    ) {
      return parsed
    }
  } catch {
    // Ignore unreadable or missing state and fall back to defaults.
  }

  return { ...defaultWindowState }
}

function persistWindowState(statePath, browserWindow) {
  const bounds = browserWindow.getBounds()
  fs.writeFileSync(statePath, JSON.stringify(bounds, null, 2))
}

module.exports = {
  defaultWindowState,
  loadWindowState,
  persistWindowState,
}
