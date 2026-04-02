import type { SkillTreeKind } from './skill-bundle-editor'

type SkillEditorLayoutControllerInput = {
  getOpenFilePaths: () => string[]
  setOpenFilePaths: (value: string[]) => void
  setSelectedFilePath: (value: string | null) => void
  setSelectedTreePath: (value: string | null) => void
  setSelectedTreeKind: (value: SkillTreeKind | null) => void
  setDragging: (value: boolean) => void
  getDragging: () => boolean
  setDragStartX: (value: number) => void
  getDragStartX: () => number
  setDragStartWidth: (value: number) => void
  getDragStartWidth: () => number
  getAssistantWidth: () => number
  setAssistantWidth: (value: number) => void
  minWidth: number
  maxWidth: number
}

export function createSkillEditorPageLayoutController(input: SkillEditorLayoutControllerInput) {
  return {
    selectFile(path: string) {
      input.setSelectedFilePath(path)
      input.setSelectedTreePath(path)
      input.setSelectedTreeKind('file')
      if (!input.getOpenFilePaths().includes(path)) {
        input.setOpenFilePaths([...input.getOpenFilePaths(), path])
      }
    },
    selectTreeNode(path: string, kind: SkillTreeKind) {
      input.setSelectedTreePath(path)
      input.setSelectedTreeKind(kind)
      if (kind === 'file') this.selectFile(path)
    },
    handleDragStart(event: PointerEvent) {
      input.setDragging(true)
      input.setDragStartX(event.clientX)
      input.setDragStartWidth(input.getAssistantWidth())
      ;(event.target as HTMLElement).setPointerCapture(event.pointerId)
    },
    handleDragMove(event: PointerEvent) {
      if (!input.getDragging()) return

      input.setAssistantWidth(
        Math.min(
          input.maxWidth,
          Math.max(
            input.minWidth,
            input.getDragStartWidth() + input.getDragStartX() - event.clientX,
          ),
        ),
      )
    },
    handleDragEnd() {
      input.setDragging(false)
    },
  }
}
