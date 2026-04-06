function registerDesktopIpc({ ipcMain, controller, shell, paths }) {
  const handlers = [
    ['desktop:get-runtime-state', () => controller.getRuntimeState()],
    ['desktop:restart-service', () => controller.restartService()],
    ['desktop:open-logs-directory', () => shell.openPath(paths.openaseLogsDir)],
    ['desktop:open-data-directory', () => shell.openPath(paths.openaseHomeDir)],
    ['desktop:open-desktop-guide', () => shell.openPath(controller.getDesktopGuidePath())],
    ['desktop:quit', () => controller.quitApplication()],
  ]

  for (const [channel, handler] of handlers) {
    ipcMain.handle(channel, handler)
  }

  return () => {
    for (const [channel] of handlers) {
      ipcMain.removeHandler(channel)
    }
  }
}

module.exports = {
  registerDesktopIpc,
}
