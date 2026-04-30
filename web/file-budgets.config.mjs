function matchesPattern(pattern) {
  return (filePath) => pattern.test(filePath)
}

function isDeclarationFile(filePath) {
  return filePath.endsWith('.d.ts')
}

function isScriptModule(filePath) {
  return /\.(ts|js|mjs|cjs)$/.test(filePath) && !isDeclarationFile(filePath)
}

function isSvelteComponent(filePath) {
  return filePath.endsWith('.svelte')
}

function defineBudgetCategory({
  key,
  name,
  softLimit,
  hardLimit,
  match,
  eslintFiles,
  eslintIgnores = [],
}) {
  return { key, name, softLimit, hardLimit, match, eslintFiles, eslintIgnores }
}

const isRoutePage = matchesPattern(
  /^src\/routes\/.+\/\+page\.svelte$|^src\/routes\/\+page\.svelte$/,
)
const isRouteLayout = matchesPattern(
  /^src\/routes\/.+\/\+layout\.svelte$|^src\/routes\/\+layout\.svelte$/,
)
const isRouteModule = (filePath) =>
  isScriptModule(filePath) &&
  (matchesPattern(/^src\/routes\/.+\/\+(page|layout|server)\.(ts|js)$/)(filePath) ||
    matchesPattern(/^src\/routes\/\+(page|layout|server)\.(ts|js)$/)(filePath) ||
    matchesPattern(/^src\/hooks(\.server|\.client)?\.(ts|js)$/)(filePath))
const isFeatureTestModule = (filePath) =>
  isScriptModule(filePath) &&
  matchesPattern(/^src\/lib\/features\/.+\.test\.(ts|js|mjs|cjs)$/)(filePath)
const isFeatureStateModule = (filePath) =>
  !isDeclarationFile(filePath) &&
  matchesPattern(/^src\/lib\/features\/.+\.svelte\.(ts|js)$/)(filePath)
const isFeatureComponent = (filePath) =>
  isSvelteComponent(filePath) && matchesPattern(/^src\/lib\/features\/.+\.svelte$/)(filePath)
const isFeatureModule = (filePath) =>
  isScriptModule(filePath) && matchesPattern(/^src\/lib\/features\/.+\.(ts|js|mjs|cjs)$/)(filePath)
const isTestingSupportModule = (filePath) =>
  isScriptModule(filePath) && matchesPattern(/^src\/lib\/testing\/.+\.(ts|js|mjs|cjs)$/)(filePath)
const isLayoutComponent = (filePath) =>
  isSvelteComponent(filePath) &&
  matchesPattern(/^src\/lib\/components\/layout\/.+\.svelte$/)(filePath)
const isLayoutSupportModule = (filePath) =>
  isScriptModule(filePath) &&
  matchesPattern(/^src\/lib\/components\/layout\/.+\.(ts|js|mjs|cjs)$/)(filePath)
const isUiPrimitive = (filePath) =>
  isSvelteComponent(filePath) && matchesPattern(/^src\/lib\/components\/ui\/.+\.svelte$/)(filePath)
const isSharedComponent = (filePath) =>
  isSvelteComponent(filePath) &&
  matchesPattern(/^src\/lib\/components\/.+\.svelte$/)(filePath) &&
  !isLayoutComponent(filePath) &&
  !isUiPrimitive(filePath)
const isStoreModule = (filePath) =>
  isScriptModule(filePath) && matchesPattern(/^src\/lib\/stores\/.+\.(ts|js|mjs|cjs)$/)(filePath)
const isApiModule = (filePath) =>
  isScriptModule(filePath) && matchesPattern(/^src\/lib\/api\/.+\.(ts|js|mjs|cjs)$/)(filePath)
const isSharedLibraryModule = (filePath) =>
  isScriptModule(filePath) &&
  matchesPattern(/^src\/lib\/.+\.(ts|js|mjs|cjs)$/)(filePath) &&
  !isFeatureModule(filePath) &&
  !isFeatureTestModule(filePath) &&
  !isFeatureStateModule(filePath) &&
  !isTestingSupportModule(filePath) &&
  !isLayoutSupportModule(filePath) &&
  !isStoreModule(filePath) &&
  !isApiModule(filePath)
const isTestHarnessModule = (filePath) =>
  isScriptModule(filePath) && matchesPattern(/^src\/test\/.+\.(ts|js|mjs|cjs)$/)(filePath)

export const fileBudgetCoverageIgnoreRules = [
  {
    name: 'Type declarations',
    match: isDeclarationFile,
  },
]

export function isBudgetCoverageIgnoredFile(filePath) {
  return fileBudgetCoverageIgnoreRules.some((rule) => rule.match(filePath))
}

export function isBudgetTrackedSourceFile(filePath) {
  return (
    !isBudgetCoverageIgnoredFile(filePath) &&
    (isScriptModule(filePath) || isSvelteComponent(filePath))
  )
}

export const fileBudgetCategories = [
  defineBudgetCategory({
    key: 'routePage',
    name: 'Route pages',
    softLimit: 150,
    hardLimit: 250,
    match: isRoutePage,
    eslintFiles: ['src/routes/**/+page.svelte'],
  }),
  defineBudgetCategory({
    key: 'routeLayout',
    name: 'Route layouts',
    softLimit: 180,
    hardLimit: 300,
    match: isRouteLayout,
    eslintFiles: ['src/routes/**/+layout.svelte'],
  }),
  defineBudgetCategory({
    key: 'routeModule',
    name: 'Route runtime modules',
    softLimit: 80,
    hardLimit: 160,
    match: isRouteModule,
    eslintFiles: ['src/routes/**/*.{js,ts,mjs,cjs}', 'src/hooks*.{js,ts,mjs,cjs}'],
    eslintIgnores: ['**/*.d.ts'],
  }),
  defineBudgetCategory({
    key: 'featureTest',
    name: 'Feature tests',
    softLimit: 300,
    hardLimit: 650,
    match: isFeatureTestModule,
    eslintFiles: ['src/lib/features/**/*.test.{js,ts,mjs,cjs}'],
  }),
  defineBudgetCategory({
    key: 'featureStateModule',
    name: 'Feature state modules',
    softLimit: 250,
    hardLimit: 350,
    match: isFeatureStateModule,
    eslintFiles: ['src/lib/features/**/*.svelte.{ts,js}'],
  }),
  defineBudgetCategory({
    key: 'featureComponent',
    name: 'Feature components',
    softLimit: 200,
    hardLimit: 350,
    match: isFeatureComponent,
    eslintFiles: ['src/lib/features/**/*.svelte'],
  }),
  defineBudgetCategory({
    key: 'featureModule',
    name: 'Feature modules',
    softLimit: 200,
    hardLimit: 325,
    match: (filePath) =>
      isFeatureModule(filePath) &&
      !isFeatureTestModule(filePath) &&
      !isFeatureStateModule(filePath),
    eslintFiles: ['src/lib/features/**/*.{js,ts,mjs,cjs}'],
    eslintIgnores: [
      'src/lib/features/**/*.test.{js,ts,mjs,cjs}',
      'src/lib/features/**/*.svelte.{ts,js}',
    ],
  }),
  defineBudgetCategory({
    key: 'testingSupportModule',
    name: 'Testing support modules',
    softLimit: 350,
    hardLimit: 650,
    match: isTestingSupportModule,
    eslintFiles: ['src/lib/testing/**/*.{js,ts,mjs,cjs}'],
  }),
  defineBudgetCategory({
    key: 'layoutComponent',
    name: 'Layout components',
    softLimit: 200,
    hardLimit: 300,
    match: isLayoutComponent,
    eslintFiles: ['src/lib/components/layout/**/*.svelte'],
  }),
  defineBudgetCategory({
    key: 'layoutSupportModule',
    name: 'Layout support modules',
    softLimit: 200,
    hardLimit: 350,
    match: isLayoutSupportModule,
    eslintFiles: ['src/lib/components/layout/**/*.{js,ts,mjs,cjs}'],
    eslintIgnores: ['**/*.d.ts'],
  }),
  defineBudgetCategory({
    key: 'uiPrimitive',
    name: 'UI primitives',
    softLimit: 150,
    hardLimit: 250,
    match: isUiPrimitive,
    eslintFiles: ['src/lib/components/ui/**/*.svelte'],
  }),
  defineBudgetCategory({
    key: 'sharedComponent',
    name: 'Shared components',
    softLimit: 250,
    hardLimit: 800,
    match: isSharedComponent,
    eslintFiles: ['src/lib/components/**/*.svelte'],
    eslintIgnores: ['src/lib/components/layout/**/*.svelte', 'src/lib/components/ui/**/*.svelte'],
  }),
  defineBudgetCategory({
    key: 'storeModule',
    name: 'Store modules',
    softLimit: 250,
    hardLimit: 350,
    match: isStoreModule,
    eslintFiles: ['src/lib/stores/**/*.{js,ts,mjs,cjs}'],
    eslintIgnores: ['**/*.d.ts'],
  }),
  defineBudgetCategory({
    key: 'apiModule',
    name: 'API boundary modules',
    softLimit: 500,
    hardLimit: 2500,
    match: isApiModule,
    eslintFiles: ['src/lib/api/**/*.{js,ts,mjs,cjs}'],
    eslintIgnores: ['src/lib/api/generated/**', '**/*.d.ts'],
  }),
  defineBudgetCategory({
    key: 'sharedLibraryModule',
    name: 'Shared library modules',
    softLimit: 200,
    hardLimit: 325,
    match: isSharedLibraryModule,
    eslintFiles: ['src/lib/**/*.{js,ts,mjs,cjs}'],
    eslintIgnores: [
      'src/lib/features/**/*.{js,ts,mjs,cjs}',
      'src/lib/testing/**/*.{js,ts,mjs,cjs}',
      'src/lib/components/layout/**/*.{js,ts,mjs,cjs}',
      'src/lib/stores/**/*.{js,ts,mjs,cjs}',
      'src/lib/api/**/*.{js,ts,mjs,cjs}',
      '**/*.d.ts',
    ],
  }),
  defineBudgetCategory({
    key: 'testHarnessModule',
    name: 'Test harness modules',
    softLimit: 150,
    hardLimit: 250,
    match: isTestHarnessModule,
    eslintFiles: ['src/test/**/*.{js,ts,mjs,cjs}'],
    eslintIgnores: ['**/*.d.ts'],
  }),
]

export const fileBudgetLimits = Object.fromEntries(
  fileBudgetCategories.map(({ key, softLimit, hardLimit }) => [
    key,
    { soft: softLimit, hard: hardLimit },
  ]),
)

export const fileBudgetRules = fileBudgetCategories.map(
  ({ name, match, softLimit, hardLimit }) => ({
    name,
    match,
    softLimit,
    hardLimit,
  }),
)

// `lint:structure` uses first-match wins while ESLint uses last-match wins, so
// the shared categories are reversed here to keep future specific overrides authoritative.
export const eslintFileBudgetOverrides = [...fileBudgetCategories]
  .reverse()
  .map(({ eslintFiles, eslintIgnores, hardLimit }) => ({
    files: eslintFiles,
    ignores: eslintIgnores,
    hardLimit,
  }))
