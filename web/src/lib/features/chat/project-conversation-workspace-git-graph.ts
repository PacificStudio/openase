import type { ProjectConversationWorkspaceGitGraphCommit } from '$lib/api/chat'

export type WorkspaceGitGraphRow = {
  commit: ProjectConversationWorkspaceGitGraphCommit
  column: number
  laneCount: number
  activeBefore: number[]
  activeAfter: number[]
  parentColumns: number[]
}

export function buildWorkspaceGitGraphRows(
  commits: ProjectConversationWorkspaceGitGraphCommit[],
): WorkspaceGitGraphRow[] {
  const rows: WorkspaceGitGraphRow[] = []
  let active: string[] = []

  for (const commit of commits) {
    let column = active.indexOf(commit.commitId)
    if (column === -1) {
      active = [commit.commitId, ...active]
      column = 0
    }

    const activeBefore = active.map((_, index) => index)
    const nextActive = [...active]
    nextActive.splice(column, 1)

    const uniqueParents = commit.parentIds.filter((parentId, index) => {
      return parentId.trim() !== '' && commit.parentIds.indexOf(parentId) === index
    })
    if (uniqueParents.length > 0) {
      nextActive.splice(column, 0, ...uniqueParents)
    }

    const dedupedActive: string[] = []
    for (const ref of nextActive) {
      if (!dedupedActive.includes(ref)) {
        dedupedActive.push(ref)
      }
    }

    const parentColumns = uniqueParents
      .map((parentId) => dedupedActive.indexOf(parentId))
      .filter((nextColumn) => nextColumn >= 0)
    const activeAfter = dedupedActive.map((_, index) => index)

    rows.push({
      commit,
      column,
      laneCount: Math.max(activeBefore.length, activeAfter.length, column + 1),
      activeBefore,
      activeAfter,
      parentColumns,
    })

    active = dedupedActive
  }

  return rows
}
