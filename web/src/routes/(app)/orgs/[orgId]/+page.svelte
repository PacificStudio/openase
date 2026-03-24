<script lang="ts">
  import OrganizationCreationLanes from '$lib/features/catalog-creation/components/organization-creation-lanes.svelte'
  import OrganizationDashboardStats from '$lib/features/catalog-creation/components/organization-dashboard-stats.svelte'
  import OrganizationProjectGrid from '$lib/features/catalog-creation/components/organization-project-grid.svelte'
  import OrganizationSettingsPanel from '$lib/features/catalog-creation/components/organization-settings-panel.svelte'
  import { appStore } from '$lib/stores/app.svelte'
  import { organizationPath } from '$lib/stores/app-context'
  import type { PageData } from './$types'

  let { data }: { data: PageData } = $props()

  const currentOrg = $derived(appStore.currentOrg ?? data.currentOrg),
    projects = $derived(data.projects),
    providers = $derived(data.providers)
</script>

<svelte:head>
  <title>{currentOrg?.name ?? 'Organization'} - OpenASE</title>
</svelte:head>

<div class="mx-auto flex w-full max-w-6xl flex-col gap-6 px-6 py-6">
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
      <h1 class="text-foreground text-2xl font-semibold">Dashboard</h1>
      <p class="text-muted-foreground mt-1 text-sm">
        Organization overview and project entry points for the current workspace.
      </p>
    </div>
  </div>

  <OrganizationDashboardStats
    orgName={currentOrg?.name ?? 'Unknown org'}
    projectCount={projects.length}
    providerCount={providers.length}
  />

  {#if currentOrg}
    <OrganizationSettingsPanel organization={currentOrg} {providers} />
  {/if}

  <OrganizationCreationLanes
    orgId={currentOrg?.id ?? null}
    defaultProviderId={currentOrg?.default_agent_provider_id ?? null}
    {projects}
    {providers}
  />

  <section class="space-y-4">
    <div>
      <h2 class="text-foreground text-lg font-semibold">Projects</h2>
      <p class="text-muted-foreground mt-1 text-sm">
        Use direct links or the top-bar switcher to move between projects.
      </p>
    </div>

    <OrganizationProjectGrid orgId={currentOrg?.id ?? null} {projects} />
  </section>
</div>
