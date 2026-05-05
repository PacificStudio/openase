import fs from 'node:fs'
import path from 'node:path'

import { describe, expect, it } from 'vitest'

import {
  fileBudgetRules,
  isBudgetCoverageIgnoredFile,
  isBudgetTrackedSourceFile,
} from '../../file-budgets.config.mjs'

const webRoot = path.resolve(import.meta.dirname, '../..')
const sourceRoot = path.join(webRoot, 'src')

describe('file budget coverage', () => {
  it('covers every tracked frontend source file or explicitly ignores it', () => {
    const uncovered = walkFiles(sourceRoot)
      .map(toRepoPath)
      .filter((filePath) => isBudgetTrackedSourceFile(filePath))
      .filter((filePath) => !fileBudgetRules.some((rule) => rule.match(filePath)))

    expect(uncovered).toEqual([])
  })

  it('keeps declaration files outside the tracked runtime budgets', () => {
    expect(isBudgetCoverageIgnoredFile('src/lib/api/generated/openapi.d.ts')).toBe(true)
    expect(isBudgetTrackedSourceFile('src/lib/api/generated/openapi.d.ts')).toBe(false)
  })

  it('tracks the file families that used to bypass lint:structure', () => {
    for (const filePath of [
      'src/hooks.server.ts',
      'src/lib/api/openase.ts',
      'src/lib/components/code/code-editor.svelte',
      'src/lib/stores/app.svelte.ts',
      'src/test/setup.ts',
    ]) {
      expect(isBudgetTrackedSourceFile(filePath)).toBe(true)
      expect(fileBudgetRules.some((rule) => rule.match(filePath))).toBe(true)
    }
  })
})

function walkFiles(directoryPath: string): string[] {
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

function toRepoPath(filePath: string): string {
  return path.relative(webRoot, filePath).split(path.sep).join('/')
}
