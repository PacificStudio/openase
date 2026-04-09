function enforceSingleInstanceLock(app, onSecondInstance) {
  const hasLock = app.requestSingleInstanceLock()
  if (!hasLock) {
    app.quit()
    return false
  }

  if (typeof onSecondInstance === 'function') {
    app.on('second-instance', onSecondInstance)
  }

  return true
}

module.exports = {
  enforceSingleInstanceLock,
}
