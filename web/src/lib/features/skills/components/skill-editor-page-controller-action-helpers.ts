import { encodeUTF8Base64 } from './skill-bundle-editor'
import type { SkillEditorPageControllerActionsState } from './skill-editor-page-controller-actions'

export function replaceOpenPathPrefix(
  state: SkillEditorPageControllerActionsState,
  previousPath: string,
  nextPath: string,
) {
  state.setOpenFilePaths(
    state
      .getOpenFilePaths()
      .map((path) =>
        path === previousPath || path.startsWith(`${previousPath}/`)
          ? `${nextPath}${path.slice(previousPath.length)}`
          : path,
      ),
  )
  const selectedFilePath = state.getSelectedFilePath()
  if (
    selectedFilePath &&
    (selectedFilePath === previousPath || selectedFilePath.startsWith(`${previousPath}/`))
  ) {
    state.setSelectedFilePath(`${nextPath}${selectedFilePath.slice(previousPath.length)}`)
  }
  const selectedTreePath = state.getSelectedTreePath()
  if (
    selectedTreePath &&
    (selectedTreePath === previousPath || selectedTreePath.startsWith(`${previousPath}/`))
  ) {
    state.setSelectedTreePath(`${nextPath}${selectedTreePath.slice(previousPath.length)}`)
  }
}

export function removeOpenPathsUnder(
  state: SkillEditorPageControllerActionsState,
  targetPath: string,
) {
  state.setOpenFilePaths(
    state
      .getOpenFilePaths()
      .filter((path) => path !== targetPath && !path.startsWith(`${targetPath}/`)),
  )
  const selectedFilePath = state.getSelectedFilePath()
  if (
    selectedFilePath &&
    (selectedFilePath === targetPath || selectedFilePath.startsWith(`${targetPath}/`))
  ) {
    state.setSelectedFilePath(state.getOpenFilePaths().at(-1) ?? null)
  }
  const selectedTreePath = state.getSelectedTreePath()
  if (
    selectedTreePath &&
    (selectedTreePath === targetPath || selectedTreePath.startsWith(`${targetPath}/`))
  ) {
    state.setSelectedTreePath(state.getSelectedFilePath())
    state.setSelectedTreeKind(state.getSelectedFilePath() ? 'file' : null)
  }
}

export function currentEntrypointContent(state: SkillEditorPageControllerActionsState) {
  return state.getDraftFiles().find((file) => file.path === 'SKILL.md')?.content ?? ''
}

export function buildBundleRequestFiles(state: SkillEditorPageControllerActionsState) {
  return state.getDraftFiles().map((file) => ({
    path: file.path,
    content_base64: file.content_base64 ?? encodeUTF8Base64(file.content ?? ''),
    media_type: file.media_type,
    is_executable: file.is_executable,
  }))
}
