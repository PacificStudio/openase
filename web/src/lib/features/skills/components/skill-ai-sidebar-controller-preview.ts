import type { SkillFile } from '$lib/api/contracts'
import type { SkillRefinementResultPayload } from '$lib/api/skill-refinement'
import {
  buildDiffPreview,
  type DiffPreview,
  type SkillSuggestion,
} from '$lib/features/skills/assistant'

export type SkillAISuggestionPreviewItem = {
  path: string
  preview: DiffPreview
}

export function createSkillAISuggestion(
  result: SkillRefinementResultPayload | null,
): SkillSuggestion | null {
  const textPreviewFiles =
    result?.candidateFiles.filter(
      (file) => file.encoding === 'utf8' && typeof file.content === 'string',
    ) ?? []
  if (result?.status !== 'verified' || textPreviewFiles.length === 0) return null

  return {
    summary: result.transcriptSummary || 'Codex verified this draft bundle.',
    files: textPreviewFiles.map((file) => ({
      path: file.path,
      content: file.content ?? '',
    })),
  }
}

export function selectSkillAISuggestionPreviewTarget(
  suggestion: SkillSuggestion | null,
  selectedSuggestionPath: string,
) {
  return (
    suggestion?.files.find((file) => file.path === selectedSuggestionPath) ??
    suggestion?.files[0] ??
    null
  )
}

export function buildSkillAISuggestionPreview(
  files: SkillFile[],
  suggestion: SkillSuggestion | null,
  selectedSuggestionPath: string,
): DiffPreview | null {
  const previewTarget = selectSkillAISuggestionPreviewTarget(suggestion, selectedSuggestionPath)
  if (!previewTarget) return null
  return buildDiffPreview(
    files.find((file) => file.path === previewTarget.path)?.content ?? '',
    previewTarget.content,
  )
}

export function buildSkillAISuggestionPreviewList(
  files: SkillFile[],
  suggestion: SkillSuggestion | null,
): SkillAISuggestionPreviewItem[] {
  return (
    suggestion?.files.map((file) => ({
      path: file.path,
      preview: buildDiffPreview(
        files.find((current) => current.path === file.path)?.content ?? '',
        file.content,
      ),
    })) ?? []
  )
}

export function isSkillAISuggestionAlreadyApplied(
  result: SkillRefinementResultPayload | null,
  appliedBundleHash: string,
  previewList: SkillAISuggestionPreviewItem[],
) {
  return (
    (result?.candidateBundleHash && appliedBundleHash === result.candidateBundleHash) ||
    (previewList.length > 0 && previewList.every((item) => !item.preview.hasChanges))
  )
}
