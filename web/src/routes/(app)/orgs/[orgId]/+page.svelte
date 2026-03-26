<script lang="ts">
  import OrganizationProjectGrid from '$lib/features/catalog-creation/components/organization-project-grid.svelte'
  import OrganizationSettingsPanel from '$lib/features/catalog-creation/components/organization-settings-panel.svelte'
  import ProjectCreationDialog from '$lib/features/catalog-creation/components/project-creation-dialog.svelte'
  import ProviderCreationDialog from '$lib/features/catalog-creation/components/provider-creation-dialog.svelte'
  import { appStore } from '$lib/stores/app.svelte'
  import { organizationPath } from '$lib/stores/app-context'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import type { PageData } from './$types'

  let { data }: { data: PageData } = $props()

  const currentOrg = $derived(appStore.currentOrg ?? data.currentOrg),
    projects = $derived(data.projects),
    providers = $derived(data.providers)

  let showProjectDialog = $state(false)
  let showProviderDialog = $state(false)
  let settingsExpanded = $state(false)
</script>

<svelte:head>
  <title>{currentOrg?.name ?? 'Organization'} - OpenASE</title>
</svelte:head>

<div class="mx-auto flex w-full max-w-6xl flex-col gap-8 px-6 py-6">
  <!-- Header -->
  <div class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
    <div class="flex flex-col gap-2">
      <p class="text-muted-foreground text-sm">
        <a href="/" class="hover:text-foreground transition-colors">Workspace</a>
        <span class="mx-2">/</span>
        <a
          href={currentOrg ? organizationPath(currentOrg.id) : '/'}
          class="hover:text-foreground transition-colors"
        >
          {currentOrg?.name ?? 'Organization'}
        </a>
      </p>
      <div>
        <h1 class="text-foreground text-2xl font-semibold">{currentOrg?.name ?? 'Organization'}</h1>
        <p class="text-muted-foreground mt-1 text-sm">
          {projects.length}
          {projects.length === 1 ? 'project' : 'projects'} · {providers.length}
          {providers.length === 1 ? 'provider' : 'providers'}
        </p>
      </div>
    </div>

    <div class="flex gap-2">
      <Button variant="outline" onclick={() => (showProviderDialog = true)}>Add provider</Button>
      <Button onclick={() => (showProjectDialog = true)}>New project</Button>
    </div>
  </div>

  <!-- Projects -->
  <section class="space-y-4">
    <div class="flex items-center justify-between">
      <h2 class="text-foreground text-lg font-semibold">Projects</h2>
    </div>

    {#if projects.length > 0}
      <OrganizationProjectGrid orgId={currentOrg?.id ?? null} {projects} />
    {:else}
      <button
        type="button"
        class="border-border hover:border-foreground/20 hover:bg-card w-full rounded-lg border border-dashed px-4 py-12 text-center transition-colors"
        onclick={() => (showProjectDialog = true)}
      >
        <p class="text-muted-foreground text-sm">No projects yet.</p>
        <p class="text-foreground mt-1 text-sm font-medium">
          Create your first project to get started
        </p>
      </button>
    {/if}
  </section>

  <!-- Providers -->
  <section class="space-y-4">
    <div class="flex items-center justify-between">
      <h2 class="text-foreground text-lg font-semibold">Providers</h2>
      {#if providers.length > 0}
        <Button variant="ghost" size="sm" onclick={() => (showProviderDialog = true)}>
          Add provider
        </Button>
      {/if}
    </div>

    {#if providers.length > 0}
      <div class="border-border divide-border divide-y rounded-lg border">
        {#each providers as provider (provider.id)}
          <div class="flex items-center justify-between gap-4 px-4 py-3">
            <div class="flex items-center gap-3 overflow-hidden">
              <div class="min-w-0">
                <p class="text-foreground truncate text-sm font-medium">{provider.name}</p>
                <p class="text-muted-foreground truncate text-xs">
                  {provider.model_name} · {provider.adapter_type} · {provider.machine_name}
                </p>
              </div>
            </div>
            <div class="flex shrink-0 items-center gap-2">
              <Badge variant="outline">{provider.machine_status}</Badge>
              <Badge variant={provider.available ? 'secondary' : 'outline'}>
                {provider.available ? 'Available' : 'Unavailable'}
              </Badge>
              {#if currentOrg?.default_agent_provider_id === provider.id}
                <Badge variant="secondary">Default</Badge>
              {/if}
            </div>
          </div>
        {/each}
      </div>
    {:else}
      <button
        type="button"
        class="border-border hover:border-foreground/20 hover:bg-card w-full rounded-lg border border-dashed px-4 py-8 text-center transition-colors"
        onclick={() => (showProviderDialog = true)}
      >
        <p class="text-muted-foreground text-sm">No providers configured.</p>
        <p class="text-foreground mt-1 text-sm font-medium">
          Add a provider to enable agent execution
        </p>
      </button>
    {/if}
  </section>

  <!-- Organization settings (collapsible) -->
  {#if currentOrg}
    <section class="space-y-4">
      <button
        type="button"
        class="text-muted-foreground hover:text-foreground flex items-center gap-2 text-sm transition-colors"
        onclick={() => (settingsExpanded = !settingsExpanded)}
      >
        <svg
          class="h-4 w-4 transition-transform {settingsExpanded ? 'rotate-90' : ''}"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
          stroke-width="2"
        >
          <path stroke-linecap="round" stroke-linejoin="round" d="M9 5l7 7-7 7" />
        </svg>
        Organization settings
      </button>

      {#if settingsExpanded}
        <OrganizationSettingsPanel organization={currentOrg} {providers} />
      {/if}
    </section>
  {/if}
</div>

<!-- Dialogs -->
{#if currentOrg}
  <ProjectCreationDialog
    orgId={currentOrg.id}
    defaultProviderId={currentOrg.default_agent_provider_id}
    {providers}
    bind:open={showProjectDialog}
  />

  <ProviderCreationDialog orgId={currentOrg.id} bind:open={showProviderDialog} />
{/if}
