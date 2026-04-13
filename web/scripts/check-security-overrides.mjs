import fs from 'node:fs'

const expectedVersion = '4.18.1'
const packageJson = JSON.parse(fs.readFileSync(new URL('../package.json', import.meta.url), 'utf8'))
const lockfileText = fs.readFileSync(new URL('../pnpm-lock.yaml', import.meta.url), 'utf8')
const issues = []

const overrideVersion = packageJson.pnpm?.overrides?.['lodash-es']
if (overrideVersion !== expectedVersion) {
  issues.push(
    `package.json must pin pnpm.overrides["lodash-es"] to ${expectedVersion}; found ${String(overrideVersion)}.`,
  )
}

const resolvedVersions = [...lockfileText.matchAll(/^[ ]{2}lodash-es@([^:]+):/gm)].map(
  (match) => match[1],
)
if (resolvedVersions.length === 0) {
  issues.push('pnpm-lock.yaml no longer contains a resolved lodash-es package entry.')
}

const unexpectedResolvedVersions = resolvedVersions.filter((version) => version !== expectedVersion)
if (unexpectedResolvedVersions.length > 0) {
  issues.push(
    `pnpm-lock.yaml resolves unexpected lodash-es versions: ${[...new Set(unexpectedResolvedVersions)].join(', ')}.`,
  )
}

const dependencyVersions = [...lockfileText.matchAll(/^[ ]{6}lodash-es: ([^\n]+)$/gm)].map(
  (match) => match[1],
)
if (dependencyVersions.length === 0) {
  issues.push('pnpm-lock.yaml no longer contains any transitive lodash-es dependency references.')
}

const unexpectedDependencyVersions = dependencyVersions.filter(
  (version) => version !== expectedVersion,
)
if (unexpectedDependencyVersions.length > 0) {
  issues.push(
    `pnpm-lock.yaml links lodash-es through unexpected versions: ${[...new Set(unexpectedDependencyVersions)].join(', ')}.`,
  )
}

if (!lockfileText.includes(`overrides:\n  lodash-es: ${expectedVersion}`)) {
  issues.push(`pnpm-lock.yaml must record the lodash-es override at version ${expectedVersion}.`)
}

if (issues.length > 0) {
  console.error('Security override check failed:')
  for (const issue of issues) {
    console.error(`- ${issue}`)
  }
  process.exit(1)
}

console.log(
  `Security override check passed: ${dependencyVersions.length} transitive lodash-es references resolve to ${expectedVersion}.`,
)
