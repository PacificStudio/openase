<script lang="ts">
  import { OrganizationSettingsPanel, ProviderCreationDialog } from '$lib/features/catalog-creation'
  import { OrganizationProvidersSection } from '$lib/features/dashboard'
  import { OrganizationScopedSecretsPanel } from '$lib/features/settings'
  import { appStore } from '$lib/stores/app.svelte'

  let { organizationId }: { organizationId: string } = $props()

  const currentOrg = $derived(
    appStore.currentOrg?.id === organizationId ? appStore.currentOrg : null,
  )
  const providers = $derived(appStore.providers)
  let showProviderDialog = $state(false)

  $effect(() => {
    void organizationId
  })
</script>

<div class="space-y-8" data-organization-id={organizationId}>
  <OrganizationProvidersSection
    {providers}
    defaultProviderId={currentOrg?.default_agent_provider_id ?? null}
    onAddProvider={() => (showProviderDialog = true)}
  />

  {#if currentOrg}
    <OrganizationSettingsPanel organization={currentOrg} {providers} />
    <OrganizationScopedSecretsPanel organizationId={currentOrg.id} />
    <ProviderCreationDialog orgId={currentOrg.id} bind:open={showProviderDialog} />
  {/if}
</div>
