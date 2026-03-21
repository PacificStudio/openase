<script lang="ts">
  import {
    capabilityCatalog,
    capabilityStateClasses,
    capabilityStateLabel,
    type CapabilityState,
  } from '$lib/features/capabilities'
  import * as Card from '$ui/card'

  type BoundaryItem = {
    label: string
    location: string
    state: CapabilityState
    summary: string
  }

  const boundaryItems: BoundaryItem[] = [
    {
      label: 'Agent registration',
      location: '/agents',
      state: capabilityCatalog.agentRegistration.state,
      summary: capabilityCatalog.agentRegistration.summary,
    },
    {
      label: 'Provider configuration',
      location: 'Settings / Agents',
      state: capabilityCatalog.providerConfigure.state,
      summary: capabilityCatalog.providerConfigure.summary,
    },
    {
      label: 'Agent deletion',
      location: 'Settings / Agents',
      state: 'unwired',
      summary:
        'DELETE /api/v1/agents/{agentId} exists, but this settings slice does not expose an inline delete flow yet.',
    },
    {
      label: 'Runtime controls',
      location: '/agents',
      state: capabilityCatalog.agentPause.state,
      summary:
        'Pause and resume stay in the runtime console and now call the explicit agent runtime control endpoints.',
    },
    {
      label: 'Runtime output',
      location: '/agents',
      state: capabilityCatalog.agentOutput.state,
      summary: capabilityCatalog.agentOutput.summary,
    },
  ]
</script>

<Card.Root>
  <Card.Header>
    <Card.Title>Capability boundaries</Card.Title>
    <Card.Description>
      Clarify what belongs in governance settings versus the runtime console.
    </Card.Description>
  </Card.Header>
  <Card.Content class="space-y-3">
    {#each boundaryItems as item (item.label)}
      <div class="border-border rounded-md border px-3 py-3">
        <div class="flex flex-wrap items-center gap-2">
          <div class="text-foreground text-sm font-medium">{item.label}</div>
          <span
            class={`inline-flex items-center rounded-full border px-2 py-0.5 text-[11px] font-medium ${capabilityStateClasses(item.state)}`}
          >
            {capabilityStateLabel(item.state)}
          </span>
          <span class="text-muted-foreground text-xs">{item.location}</span>
        </div>
        <p class="text-muted-foreground mt-2 text-xs">{item.summary}</p>
      </div>
    {/each}
  </Card.Content>
</Card.Root>
