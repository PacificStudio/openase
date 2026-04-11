import fs from 'node:fs'
import path from 'node:path'

import { fileBudgetRules } from '../file-budgets.config.mjs'

const repoRoot = process.cwd()
const sourceRoot = path.join(repoRoot, 'src')

const files = walkFiles(sourceRoot)
const warnings = []
const failures = []

for (const absoluteFile of files) {
  const relativeFile = toRepoPath(absoluteFile)
  const budget = fileBudgetRules.find((rule) => rule.match(relativeFile))
  if (!budget) {
    continue
  }

  const lineCount = fs.readFileSync(absoluteFile, 'utf8').split(/\r?\n/).length
  if (lineCount > budget.hardLimit) {
    failures.push({ file: relativeFile, lineCount, budget })
    continue
  }

  if (lineCount > budget.softLimit) {
    warnings.push({ file: relativeFile, lineCount, budget })
  }
}

if (warnings.length > 0) {
  console.warn('Soft budget warnings:')
  for (const item of warnings) {
    console.warn(`  - ${item.file}: ${item.lineCount} lines (soft limit ${item.budget.softLimit})`)
  }
  console.warn('')
}

if (failures.length > 0) {
  console.error('File budget violations:')
  for (const item of failures) {
    console.error(`  - ${item.file}: ${item.lineCount} lines (hard limit ${item.budget.hardLimit})`)
  }
  process.exit(1)
}

process.stdout.write(`File budgets passed for ${files.length} source files.\n`)

function walkFiles(directoryPath) {
  if (!fs.existsSync(directoryPath)) {
    return []
  }

  return fs.readdirSync(directoryPath, { withFileTypes: true }).flatMap((entry) => {
    const entryPath = path.join(directoryPath, entry.name)
    if (entry.isDirectory()) {
      return walkFiles(entryPath)
    }
    return entry.isFile() ? [entryPath] : []
  })
}

function toRepoPath(filePath) {
  return path.relative(repoRoot, filePath).split(path.sep).join('/')
}
