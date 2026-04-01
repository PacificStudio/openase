import { describe, expect, it } from 'vitest'

import type { SkillFile } from '$lib/api/contracts'

import {
  addEmptyDirectory,
  buildSkillTreeEntries,
  computeDirtyPaths,
  createDraftTextFile,
  deleteDirectoryPath,
  listEmptyDirectories,
  normalizeSkillBundlePath,
  renameDirectoryPath,
  renameFilePath,
  updateDraftTextFileContent,
} from './skill-bundle-editor'

function fixtureFiles(): SkillFile[] {
  return [
    createDraftTextFile('SKILL.md', '# Deploy\n'),
    createDraftTextFile('scripts/deploy.sh', '#!/usr/bin/env bash\necho deploy\n'),
    createDraftTextFile('references/runbook.md', '# Runbook\n'),
  ]
}

describe('skill-bundle-editor', () => {
  it('normalizes bundle paths and rejects escapes', () => {
    expect(normalizeSkillBundlePath(' scripts/deploy.sh ')).toBe('scripts/deploy.sh')
    expect(() => normalizeSkillBundlePath('../escape')).toThrow('skill root')
    expect(() => normalizeSkillBundlePath('/absolute')).toThrow('relative')
  })

  it('builds a tree that includes derived and empty directories', () => {
    const files = fixtureFiles()
    const entries = buildSkillTreeEntries(files, ['assets/icons'])

    expect(entries.map((entry) => `${entry.kind}:${entry.path}`)).toEqual([
      'directory:assets',
      'directory:assets/icons',
      'directory:references',
      'file:references/runbook.md',
      'directory:scripts',
      'file:scripts/deploy.sh',
      'file:SKILL.md',
    ])
  })

  it('renames directories and updates descendant files', () => {
    const renamed = renameDirectoryPath(fixtureFiles(), ['scripts/hooks'], 'scripts', 'automation')

    expect(renamed.files.map((file) => file.path)).toEqual([
      'SKILL.md',
      'automation/deploy.sh',
      'references/runbook.md',
    ])
    expect(renamed.emptyDirectoryPaths).toEqual(['automation/hooks'])
  })

  it('tracks renamed, edited, and deleted files as dirty', () => {
    const original = fixtureFiles()
    const renamed = renameFilePath(original, 'references/runbook.md', 'references/guide.md')
    const edited = renamed.map((file) =>
      file.path === 'scripts/deploy.sh'
        ? updateDraftTextFileContent(file, '#!/usr/bin/env bash\necho release\n')
        : file,
    )

    expect(Array.from(computeDirtyPaths(original, edited)).sort()).toEqual([
      'references/guide.md',
      'references/runbook.md',
      'scripts/deploy.sh',
    ])
  })

  it('deletes directory descendants and keeps empty draft folders separate', () => {
    const withFolder = addEmptyDirectory([], fixtureFiles(), 'assets/icons')
    const deleted = deleteDirectoryPath(fixtureFiles(), withFolder, 'references')

    expect(deleted.files.map((file) => file.path)).toEqual(['SKILL.md', 'scripts/deploy.sh'])
    expect(listEmptyDirectories(deleted.files, deleted.emptyDirectoryPaths)).toEqual([
      'assets/icons',
    ])
  })
})
