import type { SkillFile } from '$lib/api/contracts'
import { createDraftTextFile } from './skill-bundle-editor-files'
import {
  defaultChildDirectoryPath,
  defaultChildFilePath,
  hasPathPrefix,
  inferSkillFileKind,
  inferSkillMediaType,
  normalizeSkillBundlePath,
  replacePathPrefix,
} from './skill-bundle-editor-paths'
import type { SkillTreeEntry, SkillTreeKind } from './skill-bundle-editor-types'

type TreeNode = {
  children: Map<string, TreeNode>
  file?: SkillFile
  kind: SkillTreeKind
  name: string
  path: string
}

function buildNode(path: string, kind: SkillTreeKind, file?: SkillFile): TreeNode {
  return {
    children: new Map(),
    file,
    kind,
    name: path.split('/').pop() ?? path,
    path,
  }
}

export function collectDirectoryPaths(
  files: SkillFile[],
  emptyDirectoryPaths: string[] = [],
): Set<string> {
  const directories = new Set<string>()
  for (const file of files) {
    const parts = file.path.split('/')
    for (let index = 1; index < parts.length; index += 1) {
      directories.add(parts.slice(0, index).join('/'))
    }
  }
  for (const directory of emptyDirectoryPaths) {
    const normalized = normalizeSkillBundlePath(directory)
    const parts = normalized.split('/')
    for (let index = 1; index <= parts.length; index += 1) {
      directories.add(parts.slice(0, index).join('/'))
    }
  }
  return directories
}

function ensureAvailablePath(
  files: SkillFile[],
  emptyDirectoryPaths: string[],
  path: string,
  skipPath = '',
) {
  const directoryPaths = collectDirectoryPaths(files, emptyDirectoryPaths)
  const existingFiles = new Set(files.map((file) => file.path))
  if (path !== skipPath && existingFiles.has(path)) {
    throw new Error(`"${path}" already exists.`)
  }
  if (path !== skipPath && directoryPaths.has(path)) {
    throw new Error(`"${path}" is already a directory.`)
  }
}

export function buildSkillTreeEntries(
  files: SkillFile[],
  emptyDirectoryPaths: string[] = [],
): SkillTreeEntry[] {
  const root: TreeNode = buildNode('', 'directory')
  const directoryPaths = Array.from(collectDirectoryPaths(files, emptyDirectoryPaths)).sort(
    (a, b) => a.localeCompare(b),
  )

  for (const directoryPath of directoryPaths) {
    const parts = directoryPath.split('/')
    let current = root
    for (let index = 0; index < parts.length; index += 1) {
      const path = parts.slice(0, index + 1).join('/')
      const existing = current.children.get(parts[index])
      if (existing) {
        current = existing
        continue
      }
      const node = buildNode(path, 'directory')
      current.children.set(parts[index], node)
      current = node
    }
  }

  for (const file of files) {
    const parts = file.path.split('/')
    let current = root
    for (let index = 0; index < parts.length - 1; index += 1) {
      const path = parts.slice(0, index + 1).join('/')
      const existing = current.children.get(parts[index]) ?? buildNode(path, 'directory')
      current.children.set(parts[index], existing)
      current = existing
    }
    current.children.set(parts.at(-1) ?? file.path, buildNode(file.path, 'file', file))
  }

  const entries: SkillTreeEntry[] = []
  const visit = (node: TreeNode, depth: number) => {
    const children = Array.from(node.children.values()).sort((left, right) => {
      if (left.kind !== right.kind) {
        return left.kind === 'directory' ? -1 : 1
      }
      return left.name.localeCompare(right.name)
    })
    for (const child of children) {
      entries.push({
        depth,
        file: child.file,
        kind: child.kind,
        name: child.name,
        path: child.path,
      })
      if (child.kind === 'directory') {
        visit(child, depth + 1)
      }
    }
  }
  visit(root, 0)
  return entries
}

export function addDraftTextFile(
  files: SkillFile[],
  emptyDirectoryPaths: string[],
  filePath: string,
  content = '',
): SkillFile[] {
  const nextFile = createDraftTextFile(filePath, content)
  ensureAvailablePath(files, emptyDirectoryPaths, nextFile.path)
  return [...files, nextFile]
}

export function addEmptyDirectory(
  emptyDirectoryPaths: string[],
  files: SkillFile[],
  directoryPath: string,
): string[] {
  const normalizedPath = normalizeSkillBundlePath(directoryPath)
  ensureAvailablePath(files, emptyDirectoryPaths, normalizedPath)
  return Array.from(new Set([...emptyDirectoryPaths, normalizedPath])).sort((left, right) =>
    left.localeCompare(right),
  )
}

export function renameFilePath(
  files: SkillFile[],
  currentPath: string,
  nextPath: string,
): SkillFile[] {
  const normalizedNextPath = normalizeSkillBundlePath(nextPath)
  if (currentPath === 'SKILL.md') {
    throw new Error('SKILL.md cannot be renamed.')
  }
  ensureAvailablePath(files, [], normalizedNextPath, currentPath)

  return files.map((file) =>
    file.path === currentPath
      ? {
          ...file,
          path: normalizedNextPath,
          file_kind: inferSkillFileKind(normalizedNextPath),
          media_type: file.media_type || inferSkillMediaType(normalizedNextPath),
        }
      : file,
  )
}

export function renameDirectoryPath(
  files: SkillFile[],
  emptyDirectoryPaths: string[],
  currentPath: string,
  nextPath: string,
): { emptyDirectoryPaths: string[]; files: SkillFile[] } {
  const normalizedNextPath = normalizeSkillBundlePath(nextPath)
  if (
    !files.some((file) => hasPathPrefix(file.path, currentPath)) &&
    !emptyDirectoryPaths.some((path) => hasPathPrefix(path, currentPath))
  ) {
    throw new Error(`"${currentPath}" does not exist.`)
  }
  if (hasPathPrefix(normalizedNextPath, currentPath)) {
    throw new Error('A directory cannot be renamed into itself.')
  }

  const unaffectedFiles = files.filter((file) => !hasPathPrefix(file.path, currentPath))
  const untouchedDirectories = emptyDirectoryPaths.filter(
    (path) => !hasPathPrefix(path, currentPath),
  )
  ensureAvailablePath(unaffectedFiles, untouchedDirectories, normalizedNextPath)

  const renamedFiles = files.map((file) => {
    if (!hasPathPrefix(file.path, currentPath)) {
      return file
    }
    const updatedPath = replacePathPrefix(file.path, currentPath, normalizedNextPath)
    return {
      ...file,
      path: updatedPath,
      file_kind: inferSkillFileKind(updatedPath),
      media_type: file.media_type || inferSkillMediaType(updatedPath),
    }
  })
  const renamedDirectories = emptyDirectoryPaths.map((path) =>
    hasPathPrefix(path, currentPath)
      ? replacePathPrefix(path, currentPath, normalizedNextPath)
      : path,
  )

  return {
    emptyDirectoryPaths: Array.from(new Set(renamedDirectories)).sort((left, right) =>
      left.localeCompare(right),
    ),
    files: renamedFiles,
  }
}

export function deleteFilePath(files: SkillFile[], filePath: string): SkillFile[] {
  if (filePath === 'SKILL.md') {
    throw new Error('SKILL.md cannot be deleted.')
  }
  return files.filter((file) => file.path !== filePath)
}

export function deleteDirectoryPath(
  files: SkillFile[],
  emptyDirectoryPaths: string[],
  directoryPath: string,
): { emptyDirectoryPaths: string[]; files: SkillFile[] } {
  return {
    emptyDirectoryPaths: emptyDirectoryPaths.filter((path) => !hasPathPrefix(path, directoryPath)),
    files: files.filter((file) => !hasPathPrefix(file.path, directoryPath)),
  }
}

export function listEmptyDirectories(files: SkillFile[], emptyDirectoryPaths: string[]): string[] {
  return emptyDirectoryPaths.filter(
    (directoryPath) => !files.some((file) => hasPathPrefix(file.path, directoryPath)),
  )
}

export { defaultChildDirectoryPath, defaultChildFilePath }
