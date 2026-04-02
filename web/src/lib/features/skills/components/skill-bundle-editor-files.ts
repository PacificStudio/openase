import type { SkillFile } from '$lib/api/contracts'
import {
  inferSkillFileKind,
  inferSkillMediaType,
  normalizeSkillBundlePath,
} from './skill-bundle-editor-paths'

const textEncoder = new TextEncoder()

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

function inferExecutable(filePath: string): boolean {
  return filePath.startsWith('scripts/') && (filePath.endsWith('.sh') || !filePath.includes('.'))
}

function fileMap(files: SkillFile[]): Map<string, SkillFile> {
  return new Map(files.map((file) => [file.path, file]))
}

function textContentForComparison(file: SkillFile): string {
  if (typeof file.content_base64 === 'string') {
    return file.content_base64
  }
  return encodeUTF8Base64(file.content ?? '')
}

export function encodeUTF8Base64(content: string): string {
  return encodeBase64(textEncoder.encode(content))
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

export function updateDraftTextFileContent(file: SkillFile, content: string): SkillFile {
  const bytes = textEncoder.encode(content)
  return {
    ...file,
    content,
    content_base64: encodeBase64(bytes),
    size_bytes: bytes.byteLength,
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
