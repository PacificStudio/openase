#!/usr/bin/env node
import { execFileSync } from 'node:child_process'
import fs from 'node:fs'
import path from 'node:path'
import process from 'node:process'

const __dirname = path.dirname(new URL(import.meta.url).pathname)
const webRoot = path.resolve(__dirname, '..')
const repoRoot = path.resolve(webRoot, '..')
const defaultConfigPath = path.join(webRoot, 'i18n-check.config.json')
const defaultBaselinePath = path.join(webRoot, 'i18n-check.baseline.json')
const LETTER_RE = /[\p{L}]/u
const INLINE_EXEMPT_RE = /i18n-exempt/
const TRANSLATION_USAGE_RE = /\b(i18nStore\.t|translate\(|pageTitle\(|localeLabel\(|labelKey\s*:)/
const FORBIDDEN_FALLBACK_RE = /\btranslateWithFallback\s*\(/
const FORBIDDEN_TRANSLATION_KEY_CAST_RE = /\bas\s+TranslationKey\b/

function parseArgs(argv) {
  const options = {
    scope: 'all',
    baseRef: process.env.OPENASE_I18N_BASE_REF || 'origin/main',
    configPath: defaultConfigPath,
    baselinePath: defaultBaselinePath,
    writeBaseline: false,
  }

  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index]
    if (arg === '--') {
      continue
    }
    if (arg === '--all') {
      options.scope = 'all'
      continue
    }
    if (arg === '--diff') {
      options.scope = 'diff'
      continue
    }
    if (arg === '--write-baseline') {
      options.writeBaseline = true
      continue
    }
    if (arg === '--base-ref') {
      options.baseRef = argv[index + 1]
      index += 1
      continue
    }
    if (arg === '--config') {
      options.configPath = path.resolve(process.cwd(), argv[index + 1])
      index += 1
      continue
    }
    if (arg === '--baseline') {
      options.baselinePath = path.resolve(process.cwd(), argv[index + 1])
      index += 1
      continue
    }
    throw new Error(`Unknown argument: ${arg}`)
  }

  return options
}

function runGit(args) {
  return execFileSync('git', args, { cwd: repoRoot, encoding: 'utf8' })
}

function readDiffWithFallback(rangeArgs, fallbackArgs) {
  try {
    return runGit(rangeArgs)
  } catch (error) {
    const stderr = error instanceof Error && 'stderr' in error ? String(error.stderr ?? '') : ''
    if (!stderr.includes('no merge base')) {
      throw error
    }
    return runGit(fallbackArgs)
  }
}

function loadConfig(configPath) {
  const raw = fs.readFileSync(configPath, 'utf8')
  const parsed = JSON.parse(raw)
  return {
    ...parsed,
    allowedLiteralRegexes: parsed.allowedLiteralPatterns.map((pattern) => new RegExp(pattern, 'u')),
  }
}

function loadBaselineCounts(baselinePath) {
  if (!fs.existsSync(baselinePath)) {
    return new Map()
  }

  const parsed = JSON.parse(fs.readFileSync(baselinePath, 'utf8'))
  const counts = new Map()
  for (const entry of parsed) {
    if (
      typeof entry?.filePath !== 'string' ||
      typeof entry?.reason !== 'string' ||
      typeof entry?.literal !== 'string'
    ) {
      continue
    }
    const count = Number(entry.count ?? 1)
    counts.set(makeBaselineKey(entry), Number.isFinite(count) && count > 0 ? count : 1)
  }
  return counts
}

function writeBaselineFile(baselinePath, offenses) {
  const counts = new Map()
  for (const offense of offenses) {
    const key = makeBaselineKey(offense)
    const existing = counts.get(key)
    if (existing) {
      existing.count += 1
    } else {
      counts.set(key, {
        filePath: offense.filePath,
        reason: offense.reason,
        literal: offense.literal,
        count: 1,
      })
    }
  }

  const baseline = [...counts.values()].sort((left, right) => {
    return (
      left.filePath.localeCompare(right.filePath) ||
      left.reason.localeCompare(right.reason) ||
      left.literal.localeCompare(right.literal)
    )
  })

  fs.writeFileSync(baselinePath, JSON.stringify(baseline, null, 2) + '\n')
}

function normalizePath(filePath) {
  return filePath.split(path.sep).join('/')
}

function shouldIgnoreFile(filePath, config) {
  return (
    !/\.([cm]?js|ts|svelte)$/.test(filePath) ||
    config.ignorePathPrefixes.some((prefix) => filePath.startsWith(prefix)) ||
    config.ignorePathSubstrings.some((part) => filePath.includes(part))
  )
}

function collectAllFiles(config) {
  const output = listSourceFiles()
  return output
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean)
    .filter((filePath) => !shouldIgnoreFile(filePath, config))
}

function listSourceFiles() {
  try {
    return execFileSync('rg', ['--files', 'web/src'], {
      cwd: repoRoot,
      encoding: 'utf8',
    })
  } catch (error) {
    const code = error instanceof Error && 'code' in error ? String(error.code ?? '') : ''
    if (code !== 'ENOENT') {
      throw error
    }
    return runGit(['ls-files', '--', 'web/src'])
  }
}

function addLineRange(target, filePath, start, count) {
  if (!filePath || count <= 0) {
    return
  }
  const normalized = normalizePath(filePath)
  const entry = target.get(normalized) ?? new Set()
  for (let line = start; line < start + count; line += 1) {
    entry.add(line)
  }
  target.set(normalized, entry)
}

function parseDiffChangedLines(diffText) {
  const changed = new Map()
  let currentFile = null

  for (const line of diffText.split(/\r?\n/)) {
    if (line.startsWith('+++ ')) {
      const nextFile = line.slice(4).trim()
      currentFile = nextFile === '/dev/null' ? null : nextFile.replace(/^b\//, '')
      continue
    }
    if (!line.startsWith('@@') || !currentFile) {
      continue
    }
    const match = line.match(/\+(\d+)(?:,(\d+))?/)
    if (!match) {
      continue
    }
    const start = Number(match[1])
    const count = match[2] ? Number(match[2]) : 1
    addLineRange(changed, currentFile, start, count)
  }

  return changed
}

function mergeChangedLines(target, source) {
  for (const [filePath, lines] of source.entries()) {
    const existing = target.get(filePath) ?? new Set()
    for (const line of lines) {
      existing.add(line)
    }
    target.set(filePath, existing)
  }
}

function collectDirtyChangedLines() {
  const changed = new Map()
  const workingTreeDiff = runGit(['diff', '--unified=0', '--no-color', '--', 'web/src'])
  mergeChangedLines(changed, parseDiffChangedLines(workingTreeDiff))

  const untracked = runGit(['ls-files', '--others', '--exclude-standard', '--', 'web/src'])
    .split(/\r?\n/)
    .map((line) => normalizePath(line.trim()))
    .filter(Boolean)

  for (const filePath of untracked) {
    const absolute = path.join(repoRoot, filePath)
    if (!fs.existsSync(absolute)) {
      continue
    }
    const lineCount = fs.readFileSync(absolute, 'utf8').split(/\r?\n/).length
    addLineRange(changed, filePath, 1, lineCount)
  }

  return changed
}

function collectScopedFiles(options, config) {
  if (options.scope === 'all') {
    return { files: collectAllFiles(config), changedLines: null }
  }

  const changed = new Map()
  const baseDiff = readDiffWithFallback(
    ['diff', '--unified=0', '--no-color', `${options.baseRef}...HEAD`, '--', 'web/src'],
    ['diff', '--unified=0', '--no-color', options.baseRef, 'HEAD', '--', 'web/src'],
  )
  mergeChangedLines(changed, parseDiffChangedLines(baseDiff))
  mergeChangedLines(changed, collectDirtyChangedLines())

  const files = [...changed.keys()].filter((filePath) => !shouldIgnoreFile(filePath, config))
  return { files: files.sort(), changedLines: changed }
}

function normalizeLiteral(value) {
  return value.replace(/\s+/g, ' ').trim()
}

function isAllowedLiteral(literal, config) {
  const normalized = normalizeLiteral(literal)
  if (!normalized || !LETTER_RE.test(normalized)) {
    return true
  }
  return config.allowedLiteralRegexes.some((pattern) => pattern.test(normalized))
}

function shouldSkipLine(line, previousLine) {
  return INLINE_EXEMPT_RE.test(line) || INLINE_EXEMPT_RE.test(previousLine)
}

function report(offenses, filePath, lineNumber, literal, reason) {
  offenses.push({ filePath, lineNumber, literal: normalizeLiteral(literal), reason })
}

function scanMarkupLine(line, filePath, lineNumber, config, offenses) {
  if (FORBIDDEN_FALLBACK_RE.test(line)) {
    report(
      offenses,
      filePath,
      lineNumber,
      'translateWithFallback(...)',
      'forbidden runtime i18n fallback',
    )
  }

  for (const attribute of config.translatableAttributes) {
    const attrRe = new RegExp(
      `\\b${attribute}\\s*=\\s*\\{?['"]([^'"]*[\\p{L}][^'"]*)['"]\\}?`,
      'gu',
    )
    for (const match of line.matchAll(attrRe)) {
      const literal = match[1] ?? ''
      if (!isAllowedLiteral(literal, config) && !TRANSLATION_USAGE_RE.test(line)) {
        report(offenses, filePath, lineNumber, literal, `hardcoded ${attribute}`)
      }
    }
  }

  const textRe = />([^<{]*[\p{L}][^<{]*)</gu
  for (const match of line.matchAll(textRe)) {
    const literal = match[1] ?? ''
    if (!isAllowedLiteral(literal, config) && !TRANSLATION_USAGE_RE.test(line)) {
      report(offenses, filePath, lineNumber, literal, 'hardcoded markup text')
    }
  }
}

function scanScriptLine(line, filePath, lineNumber, config, offenses) {
  if (FORBIDDEN_FALLBACK_RE.test(line)) {
    report(
      offenses,
      filePath,
      lineNumber,
      'translateWithFallback(...)',
      'forbidden runtime i18n fallback',
    )
  }

  if (FORBIDDEN_TRANSLATION_KEY_CAST_RE.test(line)) {
    report(
      offenses,
      filePath,
      lineNumber,
      'as TranslationKey',
      'forbidden TranslationKey cast bypass',
    )
  }

  if (TRANSLATION_USAGE_RE.test(line)) {
    return
  }

  const keyGroup = config.suspiciousPropertyNames
    .map((value) => value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'))
    .join('|')
  const propertyRe = new RegExp(
    `\\b(${keyGroup})\\b\\s*[:=]\\s*\\$state\\(\\s*['"]([^'"]*[\\p{L}][^'"]*)['"]|\\b(${keyGroup})\\b\\s*[:=]\\s*['"]([^'"]*[\\p{L}][^'"]*)['"]`,
    'u',
  )
  const match = line.match(propertyRe)
  if (!match) {
    return
  }
  const literal = match[2] ?? match[4] ?? ''
  if (!isAllowedLiteral(literal, config)) {
    report(offenses, filePath, lineNumber, literal, `hardcoded ${match[1] ?? match[3]}`)
  }
}

function nextSvelteSectionMode(line, currentMode) {
  if (currentMode === 'markup') {
    if (/<script\b/i.test(line)) {
      return 'script'
    }
    if (/<style\b/i.test(line)) {
      return 'style'
    }
    return currentMode
  }

  if (currentMode === 'script' && /<\/script>/i.test(line)) {
    return 'markup'
  }
  if (currentMode === 'style' && /<\/style>/i.test(line)) {
    return 'markup'
  }
  return currentMode
}

function scanFile(filePath, changedLines, config) {
  const absolute = path.join(repoRoot, filePath)
  const lines = fs.readFileSync(absolute, 'utf8').split(/\r?\n/)
  const offenses = []
  let svelteMode = 'markup'

  for (let index = 0; index < lines.length; index += 1) {
    const lineNumber = index + 1
    if (changedLines && !changedLines.has(lineNumber)) {
      continue
    }

    const line = lines[index]
    const previousLine = lines[index - 1] ?? ''
    if (shouldSkipLine(line, previousLine)) {
      svelteMode = filePath.endsWith('.svelte')
        ? nextSvelteSectionMode(line, svelteMode)
        : svelteMode
      continue
    }

    if (filePath.endsWith('.svelte')) {
      if (svelteMode === 'markup') {
        scanMarkupLine(line, filePath, lineNumber, config, offenses)
      } else if (svelteMode === 'script') {
        scanScriptLine(line, filePath, lineNumber, config, offenses)
      }
      svelteMode = nextSvelteSectionMode(line, svelteMode)
      continue
    }

    scanScriptLine(line, filePath, lineNumber, config, offenses)
  }

  return offenses
}

function formatOffense(offense) {
  return `${offense.filePath}:${offense.lineNumber} ${offense.reason} -> ${JSON.stringify(offense.literal)}`
}

function makeBaselineKey(entry) {
  return JSON.stringify([entry.filePath, entry.reason, entry.literal])
}

function filterBaselineOffenses(offenses, baselineCounts) {
  const usedCounts = new Map()
  const unsuppressed = []

  for (const offense of offenses) {
    const key = makeBaselineKey(offense)
    const allowedCount = baselineCounts.get(key) ?? 0
    const usedCount = usedCounts.get(key) ?? 0
    if (usedCount < allowedCount) {
      usedCounts.set(key, usedCount + 1)
      continue
    }
    unsuppressed.push(offense)
  }

  return unsuppressed
}

function main() {
  const options = parseArgs(process.argv.slice(2))
  const config = loadConfig(options.configPath)
  const scope = collectScopedFiles(options, config)

  const offenses = []
  for (const filePath of scope.files) {
    const changedLines = scope.changedLines?.get(filePath) ?? null
    offenses.push(...scanFile(filePath, changedLines, config))
  }

  if (options.writeBaseline) {
    writeBaselineFile(options.baselinePath, offenses)
    console.log(
      `Wrote i18n baseline with ${offenses.length} offenses to ${path.relative(repoRoot, options.baselinePath)}.`,
    )
    return
  }

  const baselineCounts = loadBaselineCounts(options.baselinePath)
  const unsuppressed = filterBaselineOffenses(offenses, baselineCounts)

  if (unsuppressed.length > 0) {
    console.error(
      'i18n scan failed. Route user-visible strings through the shared translation API or add a reviewed baseline/exemption.',
    )
    for (const offense of unsuppressed) {
      console.error(`- ${formatOffense(offense)}`)
    }
    process.exit(1)
  }

  const scopeLabel =
    options.scope === 'all' ? 'full source tree' : `changes since ${options.baseRef}`
  const suppressedCount = offenses.length - unsuppressed.length
  const baselineLabel = suppressedCount > 0 ? ` (${suppressedCount} baseline-suppressed)` : ''
  console.log(`i18n scan passed for ${scopeLabel}${baselineLabel}.`)
}

main()
