import { translate, type AppLocale, type TranslationKey } from '$lib/i18n'

export const boardPriorityValues = ['', 'urgent', 'high', 'medium', 'low'] as const

export type BoardPriority = (typeof boardPriorityValues)[number]
export type BoardFilterPriority = Exclude<BoardPriority, ''>

const boardFilterPriorityValues = boardPriorityValues.filter(
  (priority): priority is BoardFilterPriority => priority !== '',
)

const priorityLabelKeys = {
  '': 'priority.unset',
  urgent: 'priority.urgent',
  high: 'priority.high',
  medium: 'priority.medium',
  low: 'priority.low',
} as const satisfies Record<BoardPriority, TranslationKey>

export function formatBoardPriorityLabel(priority: BoardPriority, locale: AppLocale = 'en') {
  return translate(locale, priorityLabelKeys[priority])
}

export function parseBoardFilterPriority(value: string): BoardFilterPriority | undefined {
  return boardFilterPriorityValues.find((priority) => priority === value)
}
