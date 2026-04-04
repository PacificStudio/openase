import { describe, expect, it } from 'vitest'

import { formatBoardPriorityLabel, parseBoardFilterPriority } from './priority'

describe('board priority helpers', () => {
  it('formats an unset priority as Unset', () => {
    expect(formatBoardPriorityLabel('')).toBe('Unset')
  })

  it('parses only ranked priorities into board filter values', () => {
    expect(parseBoardFilterPriority('urgent')).toBe('urgent')
    expect(parseBoardFilterPriority('')).toBeUndefined()
    expect(parseBoardFilterPriority('p1')).toBeUndefined()
  })
})
