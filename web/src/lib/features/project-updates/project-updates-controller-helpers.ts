import { writeProjectUpdatesCache } from './project-updates-cache'
import type { ProjectUpdateThread } from './types'

export const defaultThreadPageLimit = 10

export type LoadProjectUpdatesOptions = {
  showLoading?: boolean
}

export function persistProjectUpdatesSnapshot(
  projectId: string,
  snapshot: {
    threads: ProjectUpdateThread[]
    nextCursor: string
    hasMoreThreads: boolean
    loadedMorePages: boolean
  },
) {
  writeProjectUpdatesCache(projectId, snapshot)
}
