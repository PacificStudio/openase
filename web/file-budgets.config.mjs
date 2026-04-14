export const fileBudgetLimits = {
  routePage: { soft: 150, hard: 250 },
  routeLayout: { soft: 180, hard: 300 },
  featureComponent: { soft: 200, hard: 350 },
  featureTest: { soft: 300, hard: 650 },
  featureStateModule: { soft: 250, hard: 500 },
  featureModule: { soft: 200, hard: 325 },
  layoutComponent: { soft: 200, hard: 300 },
  uiPrimitive: { soft: 150, hard: 250 },
  workspaceBrowserComponent: { soft: 300, hard: 700 },
  workspaceBrowserTest: { soft: 450, hard: 900 },
  workspaceStateModule: { soft: 350, hard: 750 },
}

function buildBudgetRule({ name, match, softLimit, hardLimit, eslintFiles }) {
  return { name, match, softLimit, hardLimit, eslintFiles }
}

export const namedFileBudgetCategories = [
  buildBudgetRule({
    name: 'Workspace browser components',
    match: (filePath) =>
      /^src\/lib\/features\/chat\/project-conversation-workspace-browser(?:-.+)?\.svelte$/.test(
        filePath,
      ),
    softLimit: fileBudgetLimits.workspaceBrowserComponent.soft,
    hardLimit: fileBudgetLimits.workspaceBrowserComponent.hard,
    eslintFiles: ['src/lib/features/chat/project-conversation-workspace-browser*.svelte'],
  }),
  buildBudgetRule({
    name: 'Workspace browser tests',
    match: (filePath) =>
      /^src\/lib\/features\/chat\/project-conversation-workspace-browser(?:-.+)?\.test\.(ts|js|mjs|cjs)$/.test(
        filePath,
      ),
    softLimit: fileBudgetLimits.workspaceBrowserTest.soft,
    hardLimit: fileBudgetLimits.workspaceBrowserTest.hard,
    eslintFiles: [
      'src/lib/features/chat/project-conversation-workspace-browser*.test.{js,ts,mjs,cjs}',
    ],
  }),
  buildBudgetRule({
    name: 'Workspace state modules',
    match: (filePath) =>
      /^src\/lib\/features\/chat\/project-conversation-workspace(?:-.+)?\.svelte\.(ts|js)$/.test(
        filePath,
      ),
    softLimit: fileBudgetLimits.workspaceStateModule.soft,
    hardLimit: fileBudgetLimits.workspaceStateModule.hard,
    eslintFiles: ['src/lib/features/chat/project-conversation-workspace-*.svelte.{ts,js}'],
  }),
]

export const eslintFileBudgetOverrides = namedFileBudgetCategories.map(
  ({ name, eslintFiles, hardLimit }) => ({
    name,
    files: eslintFiles,
    hardLimit,
  }),
)

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

export const fileBudgetRules = [
  ...namedFileBudgetCategories,
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
