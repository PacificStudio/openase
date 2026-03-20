import type { MachinePayload } from '$lib/api/contracts'
import type { MachinesPageData } from '$lib/features/machines'
import type { PageLoad } from './$types'

export const load: PageLoad = async ({ fetch, parent, depends }) => {
  depends('openase:machines-page')

  const { currentOrg } = await parent()
  if (!currentOrg) {
    return {
      orgContext: { kind: 'no-org' },
      initialMachines: [],
      initialListError: null,
    } satisfies MachinesPageData
  }

  try {
    const machinesResponse = await fetch(`/api/v1/orgs/${currentOrg.id}/machines`)
    if (!machinesResponse.ok) {
      return {
        orgContext: { kind: 'ready', org: currentOrg },
        initialMachines: [],
        initialListError: 'Failed to load machines.',
      } satisfies MachinesPageData
    }

    const machineData = (await machinesResponse.json()) as MachinePayload
    return {
      orgContext: { kind: 'ready', org: currentOrg },
      initialMachines: machineData.machines ?? [],
      initialListError: null,
    } satisfies MachinesPageData
  } catch {
    return {
      orgContext: { kind: 'ready', org: currentOrg },
      initialMachines: [],
      initialListError: 'Failed to load machines.',
    } satisfies MachinesPageData
  }
}
