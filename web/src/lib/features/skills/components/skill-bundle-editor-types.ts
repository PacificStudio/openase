import type { SkillFile } from '$lib/api/contracts'

export type SkillTreeKind = 'file' | 'directory'

export type SkillTreeEntry = {
  depth: number
  file?: SkillFile
  kind: SkillTreeKind
  name: string
  path: string
}
