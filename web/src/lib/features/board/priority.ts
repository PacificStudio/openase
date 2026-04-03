export const boardPriorityValues = ['', 'urgent', 'high', 'medium', 'low'] as const

export type BoardPriority = (typeof boardPriorityValues)[number]
export type BoardFilterPriority = Exclude<BoardPriority, ''>

const boardFilterPriorityValues = boardPriorityValues.filter(
  (priority): priority is BoardFilterPriority => priority !== '',
)

export function formatBoardPriorityLabel(priority: BoardPriority) {
  switch (priority) {
    case 'urgent':
      return 'Urgent'
    case 'high':
      return 'High'
    case 'medium':
      return 'Medium'
    case 'low':
      return 'Low'
    default:
      return 'Unset'
  }
}

export function parseBoardFilterPriority(value: string): BoardFilterPriority | undefined {
  return boardFilterPriorityValues.find((priority) => priority === value)
}
