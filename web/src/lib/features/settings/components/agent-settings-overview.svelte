<script lang="ts">
  import * as Card from '$ui/card'

  let {
    selectedDefaultProviderName,
    orgDefaultProviderName,
    agentCount,
    runningAgentCount,
  }: {
    selectedDefaultProviderName: string | null
    orgDefaultProviderName: string | null
    agentCount: number
    runningAgentCount: number
  } = $props()
</script>

<div class="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
  <Card.Root>
    <Card.Header class="pb-3">
      <Card.Description>Default provider</Card.Description>
      <Card.Title>{selectedDefaultProviderName ?? orgDefaultProviderName ?? 'Unset'}</Card.Title>
    </Card.Header>
    <Card.Content class="text-muted-foreground pt-0 text-xs">
      {#if selectedDefaultProviderName}
        Project-level override active.
      {:else if orgDefaultProviderName}
        Inheriting organization default.
      {:else}
        No default provider selected yet.
      {/if}
    </Card.Content>
  </Card.Root>

  <Card.Root>
    <Card.Header class="pb-3">
      <Card.Description>Registered agents</Card.Description>
      <Card.Title>{agentCount}</Card.Title>
    </Card.Header>
    <Card.Content class="text-muted-foreground pt-0 text-xs">
      Runtime controls stay in `/agents`.
    </Card.Content>
  </Card.Root>

  <Card.Root>
    <Card.Header class="pb-3">
      <Card.Description>Running or claimed</Card.Description>
      <Card.Title>{runningAgentCount}</Card.Title>
    </Card.Header>
    <Card.Content class="text-muted-foreground pt-0 text-xs">
      Current execution state is surfaced here for governance context only.
    </Card.Content>
  </Card.Root>
</div>
