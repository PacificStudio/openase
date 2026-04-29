import type { MockState } from './constants'
import { clone } from './helpers'
import { createInitialState } from './initial-state'
import { resetMockStreamState } from './stream-state'

let mockState = createInitialState()

export function getMockState(): MockState {
  return mockState
}

export function resetMockState() {
  mockState = createInitialState()
  resetMockStreamState()
  return clone(mockState)
}
