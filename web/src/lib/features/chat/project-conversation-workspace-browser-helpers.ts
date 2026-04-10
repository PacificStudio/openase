import type { Component } from 'svelte'
import {
  File,
  FileCode2,
  FileImage,
  FileJson2,
  FileText,
  FileTerminal,
  FileSpreadsheet,
  FileCog,
  FileArchive,
  FileMusic,
  FileVideo,
  FileBraces,
  FileType,
  Lock,
  Database,
  Settings,
} from '@lucide/svelte'
import type {
  ProjectConversationWorkspaceFileStatus,
  ProjectConversationWorkspaceRepoMetadata,
} from '$lib/api/chat'

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type IconComponent = Component<any>

export interface FileIconInfo {
  icon: IconComponent
  /** Tailwind color class that works in both light and dark themes */
  colorClass: string
}

// -- Color palette (light: 600 for contrast on white, dark: 400 for contrast on dark bg) --
const C_BLUE = 'text-sky-600 dark:text-sky-400' // TypeScript, generic code
const C_YELLOW = 'text-amber-600 dark:text-amber-400' // JavaScript / JSON
const C_GREEN = 'text-emerald-600 dark:text-emerald-400' // Vue, Python, configs
const C_ORANGE = 'text-orange-600 dark:text-orange-400' // HTML, Svelte, Rust
const C_PINK = 'text-pink-600 dark:text-pink-400' // CSS / styles
const C_PURPLE = 'text-violet-600 dark:text-violet-400' // Markdown / docs
const C_TEAL = 'text-teal-600 dark:text-teal-400' // Shell / terminal
const C_ROSE = 'text-rose-600 dark:text-rose-400' // Ruby, images
const C_CYAN = 'text-cyan-600 dark:text-cyan-400' // Go, data
const C_SLATE = 'text-slate-500 dark:text-slate-400' // Archives, lock, generic
const C_INDIGO = 'text-indigo-600 dark:text-indigo-400' // Java, Kotlin

type IconEntry = [IconComponent, string]

const EXT_ICON_MAP: Record<string, IconEntry> = {
  // TypeScript
  ts: [FileCode2, C_BLUE],
  tsx: [FileCode2, C_BLUE],
  // JavaScript
  js: [FileCode2, C_YELLOW],
  jsx: [FileCode2, C_YELLOW],
  mjs: [FileCode2, C_YELLOW],
  cjs: [FileCode2, C_YELLOW],
  // Python
  py: [FileCode2, C_GREEN],
  // Ruby
  rb: [FileCode2, C_ROSE],
  // Go
  go: [FileCode2, C_CYAN],
  // Rust
  rs: [FileCode2, C_ORANGE],
  // Java / Kotlin
  java: [FileCode2, C_INDIGO],
  kt: [FileCode2, C_INDIGO],
  // C#
  cs: [FileCode2, C_PURPLE],
  // C / C++
  cpp: [FileCode2, C_BLUE],
  c: [FileCode2, C_BLUE],
  h: [FileCode2, C_BLUE],
  hpp: [FileCode2, C_BLUE],
  // Swift
  swift: [FileCode2, C_ORANGE],
  // PHP
  php: [FileCode2, C_INDIGO],
  // Lua
  lua: [FileCode2, C_BLUE],
  // R
  r: [FileCode2, C_CYAN],
  // Scala
  scala: [FileCode2, C_ROSE],
  // Zig
  zig: [FileCode2, C_ORANGE],
  // Others
  v: [FileCode2, C_TEAL],
  dart: [FileCode2, C_CYAN],
  ex: [FileCode2, C_PURPLE],
  exs: [FileCode2, C_PURPLE],
  erl: [FileCode2, C_ROSE],
  elm: [FileCode2, C_CYAN],
  hs: [FileCode2, C_PURPLE],
  ml: [FileCode2, C_ORANGE],
  clj: [FileCode2, C_GREEN],
  // Web / markup
  html: [FileBraces, C_ORANGE],
  htm: [FileBraces, C_ORANGE],
  vue: [FileBraces, C_GREEN],
  svelte: [FileBraces, C_ORANGE],
  xml: [FileBraces, C_ORANGE],
  xsl: [FileBraces, C_ORANGE],
  xslt: [FileBraces, C_ORANGE],
  // Style
  css: [FileType, C_PINK],
  scss: [FileType, C_PINK],
  sass: [FileType, C_PINK],
  less: [FileType, C_PINK],
  styl: [FileType, C_PINK],
  // JSON / config data
  json: [FileJson2, C_YELLOW],
  jsonc: [FileJson2, C_YELLOW],
  json5: [FileJson2, C_YELLOW],
  // Markup / docs
  md: [FileText, C_PURPLE],
  mdx: [FileText, C_PURPLE],
  txt: [FileText, C_SLATE],
  rst: [FileText, C_PURPLE],
  tex: [FileText, C_PURPLE],
  adoc: [FileText, C_PURPLE],
  // Shell / terminal
  sh: [FileTerminal, C_TEAL],
  bash: [FileTerminal, C_TEAL],
  zsh: [FileTerminal, C_TEAL],
  fish: [FileTerminal, C_TEAL],
  ps1: [FileTerminal, C_BLUE],
  bat: [FileTerminal, C_TEAL],
  cmd: [FileTerminal, C_TEAL],
  // Config / settings
  yaml: [FileCog, C_GREEN],
  yml: [FileCog, C_GREEN],
  toml: [FileCog, C_GREEN],
  ini: [FileCog, C_SLATE],
  env: [FileCog, C_YELLOW],
  conf: [FileCog, C_SLATE],
  cfg: [FileCog, C_SLATE],
  properties: [FileCog, C_SLATE],
  // Data / DB
  sql: [Database, C_CYAN],
  csv: [FileSpreadsheet, C_GREEN],
  tsv: [FileSpreadsheet, C_GREEN],
  xls: [FileSpreadsheet, C_GREEN],
  xlsx: [FileSpreadsheet, C_GREEN],
  // Image
  png: [FileImage, C_ROSE],
  jpg: [FileImage, C_ROSE],
  jpeg: [FileImage, C_ROSE],
  gif: [FileImage, C_ROSE],
  svg: [FileImage, C_ORANGE],
  webp: [FileImage, C_ROSE],
  ico: [FileImage, C_ROSE],
  bmp: [FileImage, C_ROSE],
  avif: [FileImage, C_ROSE],
  // Audio
  mp3: [FileMusic, C_PINK],
  wav: [FileMusic, C_PINK],
  ogg: [FileMusic, C_PINK],
  flac: [FileMusic, C_PINK],
  aac: [FileMusic, C_PINK],
  // Video
  mp4: [FileVideo, C_PURPLE],
  mkv: [FileVideo, C_PURPLE],
  avi: [FileVideo, C_PURPLE],
  mov: [FileVideo, C_PURPLE],
  webm: [FileVideo, C_PURPLE],
  // Archives
  zip: [FileArchive, C_SLATE],
  tar: [FileArchive, C_SLATE],
  gz: [FileArchive, C_SLATE],
  bz2: [FileArchive, C_SLATE],
  xz: [FileArchive, C_SLATE],
  '7z': [FileArchive, C_SLATE],
  rar: [FileArchive, C_SLATE],
  // Lock files
  lock: [Lock, C_SLATE],
  // Settings
  editorconfig: [Settings, C_SLATE],
}

const FILENAME_ICON_MAP: Record<string, IconEntry> = {
  dockerfile: [FileCog, C_CYAN],
  makefile: [FileCog, C_TEAL],
  rakefile: [FileCog, C_ROSE],
  gemfile: [FileCog, C_ROSE],
  procfile: [FileCog, C_TEAL],
  vagrantfile: [FileCog, C_BLUE],
  justfile: [FileCog, C_TEAL],
  '.gitignore': [FileCog, C_ORANGE],
  '.gitattributes': [FileCog, C_ORANGE],
  '.gitmodules': [FileCog, C_ORANGE],
  '.dockerignore': [FileCog, C_CYAN],
  '.eslintrc': [FileCog, C_INDIGO],
  '.prettierrc': [FileCog, C_PINK],
  '.env': [FileCog, C_YELLOW],
  '.env.local': [FileCog, C_YELLOW],
  '.env.example': [FileCog, C_YELLOW],
}

const DEFAULT_ICON_INFO: FileIconInfo = { icon: File, colorClass: C_SLATE }

/**
 * Returns the appropriate lucide icon component and color class for a filename.
 */
export function fileIcon(filename: string): FileIconInfo {
  const lower = filename.toLowerCase()

  // Check exact filename matches first
  const byName = FILENAME_ICON_MAP[lower]
  if (byName) return { icon: byName[0], colorClass: byName[1] }

  // Check extension
  const dotIndex = lower.lastIndexOf('.')
  if (dotIndex >= 0) {
    const ext = lower.slice(dotIndex + 1)
    const byExt = EXT_ICON_MAP[ext]
    if (byExt) return { icon: byExt[0], colorClass: byExt[1] }
  }

  return DEFAULT_ICON_INFO
}

export function repoDirtyLabel(repo: ProjectConversationWorkspaceRepoMetadata) {
  return repo.dirty
    ? `${repo.filesChanged} file${repo.filesChanged === 1 ? '' : 's'} changed`
    : 'Clean'
}

export function formatTotals(added: number, removed: number) {
  return `+${added} -${removed}`
}

export function directorySegments(path: string) {
  return path.split('/').filter((segment) => segment.length > 0)
}

export function joinSegments(segments: string[]) {
  return segments.join('/')
}

export function statusLabel(status: ProjectConversationWorkspaceFileStatus) {
  switch (status) {
    case 'added':
      return 'A'
    case 'deleted':
      return 'D'
    case 'renamed':
      return 'R'
    case 'untracked':
      return 'U'
    default:
      return 'M'
  }
}

export function statusClass(status: ProjectConversationWorkspaceFileStatus) {
  switch (status) {
    case 'added':
    case 'untracked':
      return 'text-emerald-600 dark:text-emerald-400'
    case 'deleted':
      return 'text-rose-600 dark:text-rose-400'
    case 'renamed':
      return 'text-amber-600 dark:text-amber-400'
    default:
      return 'text-sky-600 dark:text-sky-400'
  }
}
