function buildApplicationMenu({ Menu, app, controller, shell, paths }) {
  const template = [
    {
      label: 'OpenASE',
      submenu: [
        {
          label: 'Restart Local Service',
          click: () => controller.restartService(),
        },
        {
          label: 'Open Logs Directory',
          click: () => shell.openPath(paths.openaseLogsDir),
        },
        {
          label: 'Open Data Directory',
          click: () => shell.openPath(paths.openaseHomeDir),
        },
        {
          label: 'Open Desktop Guide',
          click: () => shell.openPath(controller.getDesktopGuidePath()),
        },
        { type: 'separator' },
        { role: app.isPackaged ? 'quit' : 'close' },
      ],
    },
    {
      label: 'View',
      submenu: [{ role: 'reload' }, { role: 'toggledevtools' }],
    },
    {
      label: 'Window',
      submenu: [{ role: 'minimize' }, { role: 'zoom' }, { role: 'front' }],
    },
  ]

  return Menu.buildFromTemplate(template)
}

module.exports = {
  buildApplicationMenu,
}
