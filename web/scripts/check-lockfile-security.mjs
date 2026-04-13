import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const scriptDir = dirname(fileURLToPath(import.meta.url))
const lockfilePath = resolve(scriptDir, '../pnpm-lock.yaml')
const lockfile = readFileSync(lockfilePath, 'utf8')

const bannedEntries = [
  {
    packageName: 'picomatch',
    version: '4.0.3',
    reason:
      'GHSA-3v7f-55p6-f55p / CVE-2026-33672 allows incorrect glob matching in POSIX character classes.',
  },
]

const findings = bannedEntries.filter((entry) => {
  const packagePattern = new RegExp(`(^|\\n)\\s{2}${entry.packageName}@${entry.version}:`, 'm')
  const dependencyPattern = new RegExp(
    `(^|\\n)\\s+${entry.packageName}: ${entry.version}(\\n|$)`,
    'm',
  )
  return packagePattern.test(lockfile) || dependencyPattern.test(lockfile)
})

if (findings.length === 0) {
  process.exit(0)
}

console.error(`Found banned dependency versions in ${lockfilePath}:`)
for (const finding of findings) {
  console.error(`- ${finding.packageName}@${finding.version}: ${finding.reason}`)
}
process.exit(1)
