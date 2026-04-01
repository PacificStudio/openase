import type { Skill, SkillFile, Workflow } from '$lib/api/contracts'
import { getSkill, getSkillFiles, listSkillHistory, listWorkflows } from '$lib/api/openase'

export type SkillEditorHistoryEntry = {
  id: string
  version: number
  created_by: string
  created_at: string
}

export async function loadSkillEditorData(
  skillId: string,
  projectId?: string | null,
): Promise<{
  skill: Skill
  content: string
  files: SkillFile[]
  history: SkillEditorHistoryEntry[]
  workflows: Workflow[]
}> {
  const [detailPayload, filesPayload, historyPayload, workflowPayload] = await Promise.all([
    getSkill(skillId),
    getSkillFiles(skillId),
    listSkillHistory(skillId),
    projectId ? listWorkflows(projectId) : Promise.resolve({ workflows: [] as Workflow[] }),
  ])

  return {
    skill: detailPayload.skill,
    content: detailPayload.content,
    files: filesPayload.files,
    history: historyPayload.history,
    workflows: workflowPayload.workflows,
  }
}

export function selectInitialSkillFiles(files: SkillFile[]) {
  const entrypoint = files.find((file) => file.file_kind === 'entrypoint')
  if (entrypoint) {
    return {
      selectedFilePath: entrypoint.path,
      openFilePaths: [entrypoint.path],
    }
  }
  if (files.length > 0) {
    return {
      selectedFilePath: files[0].path,
      openFilePaths: [files[0].path],
    }
  }

  return {
    selectedFilePath: null,
    openFilePaths: [] as string[],
  }
}

export function resolveEntrypointContent(
  files: SkillFile[],
  editedContents: Map<string, string>,
  fallbackContent: string,
) {
  const entrypoint = files.find((file) => file.file_kind === 'entrypoint')
  return entrypoint
    ? (editedContents.get(entrypoint.path) ?? entrypoint.content ?? fallbackContent)
    : fallbackContent
}

export function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}
