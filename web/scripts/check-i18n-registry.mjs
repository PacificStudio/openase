#!/usr/bin/env node
import { execFileSync } from 'node:child_process'
import fs from 'node:fs'
import path from 'node:path'

const __dirname = path.dirname(new URL(import.meta.url).pathname)
const webRoot = path.resolve(__dirname, '..')
const sourceRoot = path.join(webRoot, 'src')
const i18nRoot = path.join(sourceRoot, 'lib/i18n')
const enLocalePath = path.join(i18nRoot, 'locales/en.json')
const zhLocalePath = path.join(i18nRoot, 'locales/zh.json')
const TRANSLATION_KEY_RE = /^[A-Za-z][A-Za-z0-9_.-]*\.[A-Za-z0-9_.-]+$/
const CALL_START_RE =
  /\b(?:i18nStore\.t|activityT|adminAuthT|chatT|onboardingT|projectUpdatesT|providersT|scheduledJobsT|searchT|skillsT|statusesT|ticketsT|t|translateRaw)\s*\(/g
const PROPERTY_PATTERNS = [
  /\b(?:labelKey|descriptionKey|titleKey|instructionKey|translationKey|shortDescKey|keyTraitKey)\s*:\s*'([^'\\]+)'/g,
  /\b(?:labelKey|descriptionKey|titleKey|instructionKey|translationKey|shortDescKey|keyTraitKey)\s*:\s*"([^"\\]+)"/g,
]

function listSourceFiles() {
  try {
    return execFileSync('rg', ['--files', sourceRoot], {
      cwd: webRoot,
      encoding: 'utf8',
    })
      .split(/\r?\n/)
      .map((line) => line.trim())
      .filter(Boolean)
  } catch (error) {
    const code = error instanceof Error && 'code' in error ? String(error.code ?? '') : ''
    if (code !== 'ENOENT') {
      throw error
    }
    return walkDirectory(sourceRoot)
  }
}

function walkDirectory(directory) {
  const files = []
  for (const entry of fs.readdirSync(directory, { withFileTypes: true })) {
    const entryPath = path.join(directory, entry.name)
    if (entry.isDirectory()) {
      files.push(...walkDirectory(entryPath))
      continue
    }
    if (/\.(?:[cm]?js|ts|svelte)$/.test(entry.name)) {
      files.push(entryPath)
    }
  }
  return files
}

function isTranslationKeyCandidate(value) {
  return TRANSLATION_KEY_RE.test(value)
}

function readLocaleKeys(localePath) {
  return new Set(Object.keys(JSON.parse(fs.readFileSync(localePath, 'utf8'))))
}

function readCallArgumentSlice(content, startIndex) {
  let depth = 1
  let index = startIndex

  while (index < content.length && depth > 0) {
    const char = content[index]

    if (char === "'" || char === '"') {
      const quote = char
      index += 1
      while (index < content.length) {
        const current = content[index]
        if (current === '\\') {
          index += 2
          continue
        }
        if (current === quote) {
          index += 1
          break
        }
        index += 1
      }
      continue
    }

    if (char === '`') {
      index += 1
      while (index < content.length) {
        const current = content[index]
        if (current === '\\') {
          index += 2
          continue
        }
        if (current === '$' && content[index + 1] === '{') {
          index += 2
          let templateDepth = 1
          while (index < content.length && templateDepth > 0) {
            const templateChar = content[index]
            if (templateChar === '\\') {
              index += 2
              continue
            }
            if (templateChar === '{') {
              templateDepth += 1
            } else if (templateChar === '}') {
              templateDepth -= 1
            }
            index += 1
          }
          continue
        }
        if (current === '`') {
          index += 1
          break
        }
        index += 1
      }
      continue
    }

    if (char === '(') {
      depth += 1
    } else if (char === ')') {
      depth -= 1
    }

    index += 1
  }

  return content.slice(startIndex, Math.max(startIndex, index - 1))
}

function collectSourceKeys() {
  const keys = new Set()

  for (const absolutePath of listSourceFiles()) {
    const normalized = absolutePath.split(path.sep).join('/')
    if (normalized.endsWith('src/lib/i18n/index.ts')) {
      continue
    }

    const content = fs.readFileSync(absolutePath, 'utf8')

    for (const pattern of PROPERTY_PATTERNS) {
      pattern.lastIndex = 0
      for (const match of content.matchAll(pattern)) {
        const key = match[1]?.trim()
        if (key && isTranslationKeyCandidate(key)) {
          keys.add(key)
        }
      }
    }

    CALL_START_RE.lastIndex = 0
    for (const match of content.matchAll(CALL_START_RE)) {
      const openParenIndex = match.index + match[0].lastIndexOf('(') + 1
      const callSlice = readCallArgumentSlice(content, openParenIndex)
      const firstLiteral = callSlice.match(/^\s*['"]([^'"\\]+)['"]/)
      const key = firstLiteral?.[1]?.trim()
      if (key && isTranslationKeyCandidate(key)) {
        keys.add(key)
      }
    }
  }

  return [...keys].sort((left, right) => left.localeCompare(right))
}

function printList(title, values) {
  if (values.length === 0) {
    return
  }
  console.error(`${title}:`)
  for (const value of values.slice(0, 80)) {
    console.error(`  - ${value}`)
  }
  if (values.length > 80) {
    console.error(`  ... and ${values.length - 80} more`)
  }
}

const enKeys = readLocaleKeys(enLocalePath)
const zhKeys = readLocaleKeys(zhLocalePath)
const sourceKeys = collectSourceKeys()

const missingInZh = [...enKeys].filter((key) => !zhKeys.has(key)).sort((a, b) => a.localeCompare(b))
const extraInZh = [...zhKeys].filter((key) => !enKeys.has(key)).sort((a, b) => a.localeCompare(b))
const missingFromDictionary = sourceKeys.filter((key) => !enKeys.has(key))

if (missingInZh.length > 0 || extraInZh.length > 0 || missingFromDictionary.length > 0) {
  console.error('i18n dictionary check failed.')
  printList('Missing zh translations for en keys', missingInZh)
  printList('Extra zh-only keys', extraInZh)
  printList('Source-used keys missing from web/src/lib/i18n/locales/en.json', missingFromDictionary)
  process.exit(1)
}

console.log(
  `i18n dictionary is aligned (${enKeys.size} keys, ${sourceKeys.length} source-used keys checked).`,
)
