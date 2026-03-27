import { cleanup, fireEvent, render } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import AgentSettingsInventory from './agent-settings-inventory.svelte'
import type { GovernanceAgent } from './agent-settings-model'

const idleAgent: GovernanceAgent = {
  id: 'agent-1',
  name: 'Codex Worker',
  providerName: 'Codex',
  machineName: 'Localhost',
  status: 'idle',
  runtimePhase: 'none',
  activeRunCount: 0,
  lastHeartbeat: null,
}

const runningAgent: GovernanceAgent = {
  ...idleAgent,
  status: 'running',
  runtimePhase: 'executing',
  activeRunCount: 1,
  lastHeartbeat: '2026-03-27T12:00:00Z',
}

function renderInventory(agents: GovernanceAgent[], onDelete = vi.fn()) {
  return render(AgentSettingsInventory, {
    props: {
      agents,
      agentsConsoleHref: '/agents',
      deletingAgentId: null,
      onDelete,
    },
  })
}

describe('Agent settings inventory', () => {
  beforeEach(() => {
    vi.spyOn(window, 'confirm').mockReturnValue(true)
  })

  afterEach(() => {
    cleanup()
    vi.restoreAllMocks()
  })

  it('calls the inline delete handler after confirmation', async () => {
    const onDelete = vi.fn().mockResolvedValue(undefined)

    const { getByLabelText } = renderInventory([idleAgent], onDelete)

    await fireEvent.click(getByLabelText('Delete agent Codex Worker'))

    expect(window.confirm).toHaveBeenCalledWith(
      'Delete agent "Codex Worker"? This removes the project agent definition. Existing runtime history may still block deletion.',
    )
    expect(onDelete).toHaveBeenCalledWith(
      expect.objectContaining({
        id: 'agent-1',
        name: 'Codex Worker',
      }),
    )
  })

  it('disables inline deletion while the agent still has active runs', () => {
    const { getByLabelText } = renderInventory([runningAgent])

    expect((getByLabelText('Delete agent Codex Worker') as HTMLButtonElement).disabled).toBe(true)
  })
})
