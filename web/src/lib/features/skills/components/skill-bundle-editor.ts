import type { SkillFile } from '$lib/api/contracts'

export type SkillTreeKind = 'file' | 'directory'

export type SkillTreeEntry = {
  depth: number
  file?: SkillFile
  kind: SkillTreeKind
  name: string
  path: string
}

const textEncoder = new TextEncoder()

type TreeNode = {
  children: Map<string, TreeNode>
  file?: SkillFile
  kind: SkillTreeKind
  name: string
  path: string
}

function binaryString(bytes: Uint8Array): string {
  let output = ''
  for (const byte of bytes) {
    output += String.fromCharCode(byte)
  }
  return output
}

function encodeBase64(bytes: Uint8Array): string {
  if (typeof btoa !== 'function') {
    throw new Error('Base64 encoder unavailable in this environment.')
  }
  return btoa(binaryString(bytes))
}

function textContentForComparison(file: SkillFile): string {
  if (typeof file.content_base64 === 'string') {
    return file.content_base64
  }
  return encodeUTF8Base64(file.content ?? '')
}

function parentPath(path: string): string {
  const parts = path.split('/')
  parts.pop()
  return parts.join('/')
}

function hasPathPrefix(path: string, prefix: string): boolean {
  return path === prefix || path.startsWith(`${prefix}/`)
}

function replacePathPrefix(path: string, prefix: string, replacement: string): string {
  if (path === prefix) {
    return replacement
  }
  return `${replacement}${path.slice(prefix.length)}`
}

function fileMap(files: SkillFile[]): Map<string, SkillFile> {
  return new Map(files.map((file) => [file.path, file]))
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

function buildNode(path: string, kind: SkillTreeKind, file?: SkillFile): TreeNode {
  return {
    children: new Map(),
    file,
    kind,
    name: path.split('/').pop() ?? path,
    path,
  }
}

function inferExecutable(filePath: string): boolean {
  return filePath.startsWith('scripts/') && (filePath.endsWith('.sh') || !filePath.includes('.'))
}

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

export function encodeUTF8Base64(content: string): string {
  return encodeBase64(textEncoder.encode(content))
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

export function cloneSkillFile(file: SkillFile): SkillFile {
  return {
    ...file,
    content: file.content,
    content_base64: file.content_base64,
  }
}

export function createDraftTextFile(filePath: string, content = ''): SkillFile {
  const normalizedPath = normalizeSkillBundlePath(filePath)
  const bytes = textEncoder.encode(content)
  return {
    path: normalizedPath,
    file_kind: inferSkillFileKind(normalizedPath),
    media_type: inferSkillMediaType(normalizedPath),
    encoding: 'utf8',
    is_executable: inferExecutable(normalizedPath),
    size_bytes: bytes.byteLength,
    sha256: '',
    content,
    content_base64: encodeBase64(bytes),
  }
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

export function updateDraftTextFileContent(file: SkillFile, content: string): SkillFile {
  const bytes = textEncoder.encode(content)
  return {
    ...file,
    content,
    content_base64: encodeBase64(bytes),
    size_bytes: bytes.byteLength,
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

export function computeDirtyPaths(
  originalFiles: SkillFile[],
  draftFiles: SkillFile[],
): Set<string> {
  const dirtyPaths = new Set<string>()
  const originalByPath = fileMap(originalFiles)
  const draftByPath = fileMap(draftFiles)

  for (const [path, original] of originalByPath) {
    const draft = draftByPath.get(path)
    if (!draft) {
      dirtyPaths.add(path)
      continue
    }
    if (
      original.is_executable !== draft.is_executable ||
      original.media_type !== draft.media_type ||
      original.encoding !== draft.encoding ||
      textContentForComparison(original) !== textContentForComparison(draft)
    ) {
      dirtyPaths.add(path)
    }
  }

  for (const path of draftByPath.keys()) {
    if (!originalByPath.has(path)) {
      dirtyPaths.add(path)
    }
  }

  return dirtyPaths
}

export function listEmptyDirectories(files: SkillFile[], emptyDirectoryPaths: string[]): string[] {
  return emptyDirectoryPaths.filter(
    (directoryPath) => !files.some((file) => hasPathPrefix(file.path, directoryPath)),
  )
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
