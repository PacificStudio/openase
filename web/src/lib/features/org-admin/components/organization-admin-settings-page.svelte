<script lang="ts">
  import { OrganizationSettingsPanel, ProviderCreationDialog } from '$lib/features/catalog-creation'
  import { OrganizationProvidersSection } from '$lib/features/dashboard'
  import { appStore } from '$lib/stores/app.svelte'
  import { Button } from '$ui/button'

  let { organizationId }: { organizationId: string } = $props()

  const currentOrg = $derived(appStore.currentOrg)
  const providers = $derived(appStore.providers)
  let showProviderDialog = $state(false)
</script>

<div class="space-y-8">
  <div class="space-y-4">
    <div class="flex items-center justify-between">
      <div class="text-sm font-semibold">Providers</div>
      <Button variant="outline" size="sm" onclick={() => (showProviderDialog = true)}>
        Add provider
      </Button>
    </div>
    <OrganizationProvidersSection
      {providers}
      defaultProviderId={currentOrg?.default_agent_provider_id ?? null}
      onAddProvider={() => (showProviderDialog = true)}
    />
  </div>

  {#if currentOrg}
    <OrganizationSettingsPanel organization={currentOrg} {providers} />
    <ProviderCreationDialog orgId={currentOrg.id} bind:open={showProviderDialog} />
  {/if}
</div>
