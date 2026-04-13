export const fileBudgetLimits = {
  routePage: { soft: 150, hard: 250 },
  routeLayout: { soft: 180, hard: 300 },
  featureComponent: { soft: 200, hard: 350 },
  featureTest: { soft: 300, hard: 650 },
  featureStateModule: { soft: 250, hard: 500 },
  featureModule: { soft: 200, hard: 325 },
  layoutComponent: { soft: 200, hard: 300 },
  uiPrimitive: { soft: 150, hard: 250 },
  workspaceBrowserPane: { soft: 250, hard: 350 },
  workspaceBrowserStateModule: { soft: 350, hard: 700 },
  workspaceBrowserIntegrationTest: { soft: 350, hard: 700 },
}

function isRoutePage(filePath) {
  return /^src\/routes\/.+\/\+page\.svelte$|^src\/routes\/\+page\.svelte$/.test(filePath)
}

function isRouteLayout(filePath) {
  return /^src\/routes\/.+\/\+layout\.svelte$|^src\/routes\/\+layout\.svelte$/.test(filePath)
}

function isFeatureTestModule(filePath) {
  return /^src\/lib\/features\/.+\.test\.(ts|js|mjs|cjs)$/.test(filePath)
}

function isFeatureStateModule(filePath) {
  return /^src\/lib\/features\/.+\.svelte\.(ts|js)$/.test(filePath)
}

function isFeatureComponent(filePath) {
  return /^src\/lib\/features\/.+\.svelte$/.test(filePath)
}

function isFeatureModule(filePath) {
  return /^src\/lib\/features\/.+\.(ts|js|mjs|cjs)$/.test(filePath)
}

function isLayoutComponent(filePath) {
  return /^src\/lib\/components\/layout\/.+\.svelte$/.test(filePath)
}

function isUiPrimitive(filePath) {
  return /^src\/lib\/components\/ui\/.+\.svelte$/.test(filePath)
}

function isWorkspaceBrowserPane(filePath) {
  return /^src\/lib\/features\/chat\/project-conversation-workspace-browser-(detail|pane|sidebar)\.svelte$/.test(
    filePath,
  )
}

function isWorkspaceBrowserStateModule(filePath) {
  return /^src\/lib\/features\/chat\/project-conversation-workspace-(browser-state|file-editor-state)\.svelte\.(ts|js)$/.test(
    filePath,
  )
}

function isWorkspaceBrowserIntegrationTest(filePath) {
  return /^src\/lib\/features\/chat\/project-conversation-workspace-browser-.*\.test\.(ts|js|mjs|cjs)$/.test(
    filePath,
  )
}

export const fileBudgetRules = [
  // These workspace-browser files coordinate tree, editor, and diff state as one
  // surface, so we give that recurring shape first-class budgets instead of
  // piling up one-off waivers.
  {
    name: 'Workspace browser panes',
    match: isWorkspaceBrowserPane,
    softLimit: fileBudgetLimits.workspaceBrowserPane.soft,
    hardLimit: fileBudgetLimits.workspaceBrowserPane.hard,
  },
  {
    name: 'Workspace browser state modules',
    match: isWorkspaceBrowserStateModule,
    softLimit: fileBudgetLimits.workspaceBrowserStateModule.soft,
    hardLimit: fileBudgetLimits.workspaceBrowserStateModule.hard,
  },
  {
    name: 'Workspace browser integration tests',
    match: isWorkspaceBrowserIntegrationTest,
    softLimit: fileBudgetLimits.workspaceBrowserIntegrationTest.soft,
    hardLimit: fileBudgetLimits.workspaceBrowserIntegrationTest.hard,
  },
  {
    name: 'Route pages',
    match: isRoutePage,
    softLimit: fileBudgetLimits.routePage.soft,
    hardLimit: fileBudgetLimits.routePage.hard,
  },
  {
    name: 'Route layouts',
    match: isRouteLayout,
    softLimit: fileBudgetLimits.routeLayout.soft,
    hardLimit: fileBudgetLimits.routeLayout.hard,
  },
  {
    name: 'Feature tests',
    match: isFeatureTestModule,
    softLimit: fileBudgetLimits.featureTest.soft,
    hardLimit: fileBudgetLimits.featureTest.hard,
  },
  {
    name: 'Feature state modules',
    match: isFeatureStateModule,
    softLimit: fileBudgetLimits.featureStateModule.soft,
    hardLimit: fileBudgetLimits.featureStateModule.hard,
  },
  {
    name: 'Feature components',
    match: isFeatureComponent,
    softLimit: fileBudgetLimits.featureComponent.soft,
    hardLimit: fileBudgetLimits.featureComponent.hard,
  },
  {
    name: 'Feature modules',
    match: (filePath) =>
      isFeatureModule(filePath) &&
      !isFeatureTestModule(filePath) &&
      !isFeatureStateModule(filePath),
    softLimit: fileBudgetLimits.featureModule.soft,
    hardLimit: fileBudgetLimits.featureModule.hard,
  },
  {
    name: 'Layout components',
    match: isLayoutComponent,
    softLimit: fileBudgetLimits.layoutComponent.soft,
    hardLimit: fileBudgetLimits.layoutComponent.hard,
  },
  {
    name: 'UI primitives',
    match: isUiPrimitive,
    softLimit: fileBudgetLimits.uiPrimitive.soft,
    hardLimit: fileBudgetLimits.uiPrimitive.hard,
  },
]
