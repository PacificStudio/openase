import type { SkillFile } from '$lib/api/contracts'
import type { SkillTreeKind } from './skill-bundle-editor-types'

export function normalizeSkillBundlePath(raw: string): string {
  const trimmed = raw.trim()
  if (!trimmed) {
    throw new Error('Path is required.')
  }
  if (trimmed.includes('\\')) {
    throw new Error('Path must use forward slashes.')
  }
  if (trimmed.startsWith('/')) {
    throw new Error('Path must be relative to the skill root.')
  }

  const parts = trimmed.split('/').filter((part) => part.length > 0)
  if (parts.length === 0) {
    throw new Error('Path is required.')
  }
  for (const part of parts) {
    if (part === '.' || part === '..') {
      throw new Error('Path cannot escape the skill root.')
    }
  }

  return parts.join('/')
}

export function parentPath(path: string): string {
  const parts = path.split('/')
  parts.pop()
  return parts.join('/')
}

export function hasPathPrefix(path: string, prefix: string): boolean {
  return path === prefix || path.startsWith(`${prefix}/`)
}

export function replacePathPrefix(path: string, prefix: string, replacement: string): string {
  if (path === prefix) {
    return replacement
  }
  return `${replacement}${path.slice(prefix.length)}`
}

export function inferSkillFileKind(filePath: string): SkillFile['file_kind'] {
  if (filePath === 'SKILL.md') return 'entrypoint'
  if (filePath === 'agents/openai.yaml') return 'metadata'
  if (filePath.startsWith('scripts/')) return 'script'
  if (filePath.startsWith('references/')) return 'reference'
  return 'asset'
}

export function inferSkillMediaType(filePath: string): string {
  if (filePath.endsWith('.md')) return 'text/markdown; charset=utf-8'
  if (filePath.endsWith('.sh')) return 'text/x-shellscript; charset=utf-8'
  if (filePath.endsWith('.yaml') || filePath.endsWith('.yml'))
    return 'application/yaml; charset=utf-8'
  if (filePath.endsWith('.json')) return 'application/json; charset=utf-8'
  if (filePath.endsWith('.txt')) return 'text/plain; charset=utf-8'
  return 'text/plain; charset=utf-8'
}

export function defaultChildFilePath(
  selectedPath: string | null,
  selectionKind: SkillTreeKind | null,
): string {
  if (!selectedPath) return 'notes.md'
  if (selectionKind === 'directory') return `${selectedPath}/notes.md`
  const parent = parentPath(selectedPath)
  return parent ? `${parent}/notes.md` : 'notes.md'
}

export function defaultChildDirectoryPath(
  selectedPath: string | null,
  selectionKind: SkillTreeKind | null,
): string {
  if (!selectedPath) return 'folder'
  if (selectionKind === 'directory') return `${selectedPath}/folder`
  const parent = parentPath(selectedPath)
  return parent ? `${parent}/folder` : 'folder'
}
