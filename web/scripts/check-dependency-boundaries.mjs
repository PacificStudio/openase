import fs from 'node:fs'
import path from 'node:path'
import { aliasRoots, boundaryRules } from '../dependency-boundaries.config.mjs'

const repoRoot = process.cwd()
const sourceRoot = path.join(repoRoot, 'src')
const supportedExtensions = ['.svelte', '.svelte.ts', '.svelte.js', '.ts', '.js', '.mjs', '.cjs']

const files = walkFiles(sourceRoot).filter(isSupportedSourceFile)
const violations = []
const waivedViolations = []

for (const filePath of files) {
  const relativeFile = toRepoPath(filePath)
  const contents = fs.readFileSync(filePath, 'utf8')

  for (const imported of collectImports(contents)) {
    const resolvedPath = resolveImport(relativeFile, imported.specifier)
    if (!resolvedPath) {
      continue
    }

    for (const rule of boundaryRules) {
      const fromMatch = rule.from.exec(relativeFile)
      if (!fromMatch) {
        continue
      }

      const message = evaluateRule(rule, {
        fromMatch,
        fromPath: relativeFile,
        toPath: resolvedPath,
      })
      if (!message) {
        continue
      }

      const waiverReason = rule.allowlist?.[relativeFile]
      const result = {
        file: relativeFile,
        line: imported.line,
        specifier: imported.specifier,
        target: resolvedPath,
        rule: rule.name,
        reason: message,
      }

      if (waiverReason) {
        waivedViolations.push({ ...result, waiverReason })
      } else {
        violations.push(result)
      }
    }
  }
}

if (waivedViolations.length > 0) {
  console.warn('Legacy dependency boundary waivers:')
  for (const waived of uniqueIssues(waivedViolations)) {
    console.warn(
      `  - ${waived.file}:${waived.line} imports ${waived.specifier} -> ${waived.target}`,
    )
    console.warn(`    ${waived.reason}`)
    console.warn(`    waiver: ${waived.waiverReason}`)
  }
  console.warn('')
}

if (violations.length > 0) {
  console.error('Dependency boundary violations:')
  for (const violation of uniqueIssues(violations)) {
    console.error(
      `  - ${violation.file}:${violation.line} imports ${violation.specifier} -> ${violation.target}`,
    )
    console.error(`    ${violation.reason}`)
  }
  process.exit(1)
}

process.stdout.write(`Dependency boundaries passed for ${files.length} source files.\n`)

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

function isSupportedSourceFile(filePath) {
  return supportedExtensions.some((extension) => filePath.endsWith(extension))
}

function collectImports(contents) {
  const imports = []
  const patterns = [
    /\bimport\s+(?:type\s+)?(?:[^'"]*?\sfrom\s*)?['"]([^'"]+)['"]/g,
    /\bexport\s+(?:type\s+)?[^'"]*?\sfrom\s*['"]([^'"]+)['"]/g,
    /\bimport\s*\(\s*['"]([^'"]+)['"]\s*\)/g,
  ]

  for (const pattern of patterns) {
    for (const match of contents.matchAll(pattern)) {
      const [fullMatch, specifier] = match
      const line = contents.slice(0, match.index).split('\n').length
      imports.push({ specifier, line, match: fullMatch })
    }
  }

  return imports
}

function resolveImport(fromPath, specifier) {
  if (specifier.startsWith('.')) {
    const absoluteBase = path.resolve(repoRoot, path.dirname(fromPath), specifier)
    return resolveWithExtensions(absoluteBase)
  }

  for (const [alias, aliasRoot] of Object.entries(aliasRoots)) {
    if (specifier === alias || specifier.startsWith(`${alias}/`)) {
      const suffix = specifier === alias ? '' : specifier.slice(alias.length + 1)
      const absoluteBase = path.resolve(repoRoot, aliasRoot, suffix)
      return resolveWithExtensions(absoluteBase)
    }
  }

  if (specifier.startsWith('src/')) {
    return resolveWithExtensions(path.resolve(repoRoot, specifier))
  }

  return null
}

function resolveWithExtensions(absoluteBase) {
  const candidates = [absoluteBase]

  for (const extension of supportedExtensions) {
    candidates.push(`${absoluteBase}${extension}`)
  }

  for (const extension of supportedExtensions) {
    candidates.push(path.join(absoluteBase, `index${extension}`))
  }

  for (const candidate of candidates) {
    if (fs.existsSync(candidate) && fs.statSync(candidate).isFile()) {
      return toRepoPath(candidate)
    }
  }

  return null
}

function evaluateRule(rule, context) {
  if (typeof rule.check === 'function') {
    const customReason = rule.check(context)
    if (customReason) {
      return customReason
    }
  }

  for (const entry of rule.disallow ?? []) {
    if (entry.to.test(context.toPath)) {
      return entry.reason
    }
  }

  return null
}

function uniqueIssues(issues) {
  const seen = new Set()
  return issues.filter((issue) => {
    const key = JSON.stringify(issue)
    if (seen.has(key)) {
      return false
    }
    seen.add(key)
    return true
  })
}

function toRepoPath(filePath) {
  return path.relative(repoRoot, filePath).split(path.sep).join('/')
}
