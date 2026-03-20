import type { PageLoad } from './$types'
import type { MachinePayload, Organization } from '$lib/api/contracts'
import type { MachinesPageData } from '$lib/features/machines'

type OrgResponse = {
  organizations?: Organization[]
}

export const load: PageLoad = async ({ fetch, depends }) => {
  depends('openase:machines-page')

  const orgContext = await loadOrgContext(fetch)
  if (orgContext.kind !== 'ready') {
    return {
      orgContext,
      initialMachines: [],
      initialListError: null,
    } satisfies MachinesPageData
  }

  try {
    const machinesResponse = await fetch(`/api/v1/orgs/${orgContext.org.id}/machines`)
    if (!machinesResponse.ok) {
      return {
        orgContext,
        initialMachines: [],
        initialListError: 'Failed to load machines.',
      } satisfies MachinesPageData
    }

    const machineData = (await machinesResponse.json()) as MachinePayload
    return {
      orgContext,
      initialMachines: machineData.machines ?? [],
      initialListError: null,
    } satisfies MachinesPageData
  } catch {
    return {
      orgContext,
      initialMachines: [],
      initialListError: 'Failed to load machines.',
    } satisfies MachinesPageData
  }
}

async function loadOrgContext(
  fetch: PageLoadEvent['fetch'],
): Promise<MachinesPageData['orgContext']> {
  try {
    const orgResponse = await fetch('/api/v1/orgs')
    if (!orgResponse.ok) {
      return { kind: 'error', message: 'Failed to load organizations.' }
    }

    const orgData = (await orgResponse.json()) as OrgResponse
    const org = orgData.organizations?.[0] ?? null
    if (!org) {
      return { kind: 'no-org' }
    }

    return { kind: 'ready', org }
  } catch {
    return { kind: 'error', message: 'Failed to load organizations.' }
  }
}

type PageLoadEvent = Parameters<PageLoad>[0]
