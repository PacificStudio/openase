export type ProjectUpdateStatus = 'on_track' | 'at_risk' | 'off_track'

export type ProjectUpdateComment = {
  id: string
  threadId: string
  bodyMarkdown: string
  createdBy: string
  createdAt: string
  updatedAt: string
  editedAt?: string
  editCount: number
  lastEditedBy?: string
  isDeleted: boolean
  deletedAt?: string
  deletedBy?: string
}

export type ProjectUpdateThread = {
  id: string
  projectId: string
  status: ProjectUpdateStatus
  title: string
  bodyMarkdown: string
  createdBy: string
  createdAt: string
  updatedAt: string
  editedAt?: string
  editCount: number
  lastEditedBy?: string
  isDeleted: boolean
  deletedAt?: string
  deletedBy?: string
  lastActivityAt: string
  commentCount: number
  comments: ProjectUpdateComment[]
}
