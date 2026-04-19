function matchesPattern(pattern) {
  return (filePath) => pattern.test(filePath)
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
const isFeatureTestModule = matchesPattern(/^src\/lib\/features\/.+\.test\.(ts|js|mjs|cjs)$/)
const isFeatureStateModule = matchesPattern(/^src\/lib\/features\/.+\.svelte\.(ts|js)$/)
const isFeatureComponent = matchesPattern(/^src\/lib\/features\/.+\.svelte$/)
const isFeatureModule = matchesPattern(/^src\/lib\/features\/.+\.(ts|js|mjs|cjs)$/)
const isTestingSupportModule = matchesPattern(/^src\/lib\/testing\/.+\.(ts|js|mjs|cjs)$/)
const isLayoutComponent = matchesPattern(/^src\/lib\/components\/layout\/.+\.svelte$/)
const isUiPrimitive = matchesPattern(/^src\/lib\/components\/ui\/.+\.svelte$/)

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
    hardLimit: 500,
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
    key: 'uiPrimitive',
    name: 'UI primitives',
    softLimit: 150,
    hardLimit: 250,
    match: isUiPrimitive,
    eslintFiles: ['src/lib/components/ui/**/*.svelte'],
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
