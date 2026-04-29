import fs from 'node:fs'
import path from 'node:path'

const repoRoot = process.cwd()
const packageJsonPath = path.join(repoRoot, 'package.json')
const lockfilePath = path.join(repoRoot, 'pnpm-lock.yaml')

const requiredOverrides = {
  'lodash-es': {
    requiredVersion: '4.18.1',
    staleVersion: '4.17.23',
  },
  uuid: {
    requiredVersion: '14.0.0',
    staleVersion: '11.1.0',
  },
}

const packageJson = JSON.parse(fs.readFileSync(packageJsonPath, 'utf8'))
const packageOverrides = packageJson.pnpm?.overrides ?? {}
const lockfile = fs.readFileSync(lockfilePath, 'utf8')
const problems = []

for (const [packageName, config] of Object.entries(requiredOverrides)) {
  const { requiredVersion, staleVersion } = config
  const packageOverride = packageOverrides[packageName]
  if (packageOverride !== requiredVersion) {
    problems.push(
      `package.json must pin pnpm.overrides.${packageName} to ${requiredVersion} (found ${String(packageOverride)})`,
    )
  }

  const overridePattern = new RegExp(
    `(^|\\n)overrides:\\n(?:[ \\t].*\\n)*?[ \\t]{2}${escapeRegExp(packageName)}: ${escapeRegExp(requiredVersion)}(?:\\n|$)`,
    'm',
  )
  if (!overridePattern.test(lockfile)) {
    problems.push(`pnpm-lock.yaml must record the ${packageName} override at ${requiredVersion}`)
  }

  const staleVersionPattern = new RegExp(
    `${escapeRegExp(packageName)}(?::|@)\\s*${escapeRegExp(staleVersion)}\\b|${escapeRegExp(packageName)}@${escapeRegExp(staleVersion)}:`,
    'm',
  )
  if (staleVersionPattern.test(lockfile)) {
    problems.push(`pnpm-lock.yaml still references stale ${packageName} ${staleVersion} entries`)
  }

  const resolvedVersionPattern = new RegExp(
    `${escapeRegExp(packageName)}(?::|@)\\s*${escapeRegExp(requiredVersion)}\\b`,
    'm',
  )
  if (!resolvedVersionPattern.test(lockfile)) {
    problems.push(
      `pnpm-lock.yaml does not contain resolved ${packageName} ${requiredVersion} entries`,
    )
  }
}

if (problems.length > 0) {
  console.error('Security override checks failed:')
  for (const problem of problems) {
    console.error(`- ${problem}`)
  }
  process.exit(1)
}

process.stdout.write(
  `Security overrides passed for ${Object.keys(requiredOverrides).join(', ')}.\n`,
)

function escapeRegExp(value) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}
