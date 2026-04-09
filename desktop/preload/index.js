const { contextBridge, ipcRenderer } = require('electron')

contextBridge.exposeInMainWorld('openaseDesktop', {
  getRuntimeState: () => ipcRenderer.invoke('desktop:get-runtime-state'),
  restartService: () => ipcRenderer.invoke('desktop:restart-service'),
  recheckSetup: () => ipcRenderer.invoke('desktop:recheck-setup'),
  applySetup: (payload) => ipcRenderer.invoke('desktop:apply-setup', payload),
  openLogsDirectory: () => ipcRenderer.invoke('desktop:open-logs-directory'),
  openDataDirectory: () => ipcRenderer.invoke('desktop:open-data-directory'),
  openDesktopGuide: () => ipcRenderer.invoke('desktop:open-desktop-guide'),
  quit: () => ipcRenderer.invoke('desktop:quit'),
})
