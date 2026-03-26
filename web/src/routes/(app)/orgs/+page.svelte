<script lang="ts">
  import OrganizationCreationDialog from '$lib/features/catalog-creation/components/organization-creation-dialog.svelte'
  import OrganizationDeleteDialog from '$lib/features/catalog-creation/components/organization-delete-dialog.svelte'
  import { organizationPath } from '$lib/stores/app-context'
  import { Button } from '$ui/button'
  import type { PageData } from './$types'

  let { data }: { data: PageData } = $props()

  type OrganizationItem = PageData['organizations'][number]

  const organizations = $derived(data.organizations)

  let showCreateDialog = $state(false)
  let deleteTarget = $state<OrganizationItem | null>(null)
  let showDeleteDialog = $state(false)

  function openDelete(org: OrganizationItem) {
    deleteTarget = org
    showDeleteDialog = true
  }
</script>

<svelte:head>
  <title>Organizations - OpenASE</title>
</svelte:head>

<div class="mx-auto flex w-full max-w-6xl flex-col gap-8 px-6 py-6">
  <!-- Header -->
  <div class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
    <div>
      <h1 class="text-foreground text-2xl font-semibold">Organizations</h1>
      <p class="text-muted-foreground mt-1 text-sm">
        Manage workspace organizations. Each organization scopes projects, providers, and machines.
        Archived organizations are removed from the active workspace.
      </p>
    </div>

    <Button onclick={() => (showCreateDialog = true)}>New organization</Button>
  </div>

  <!-- Org list -->
  {#if organizations.length > 0}
    <div class="border-border divide-border divide-y rounded-lg border">
      {#each organizations as org (org.id)}
        <div class="flex items-center justify-between gap-4 px-5 py-4">
          <div class="min-w-0">
            <a
              href={organizationPath(org.id)}
              class="text-foreground truncate text-sm font-medium hover:underline"
            >
              {org.name}
            </a>
            <p class="text-muted-foreground mt-0.5 truncate text-xs">
              {org.slug}
            </p>
          </div>

          <div class="flex shrink-0 items-center gap-2">
            <Button variant="ghost" size="sm" href={organizationPath(org.id)}>Open</Button>
            <Button
              variant="ghost"
              size="sm"
              class="text-destructive hover:text-destructive"
              onclick={() => openDelete(org)}
            >
              Archive
            </Button>
          </div>
        </div>
      {/each}
    </div>
  {:else}
    <button
      type="button"
      class="border-border hover:border-foreground/20 hover:bg-card w-full rounded-lg border border-dashed px-4 py-12 text-center transition-colors"
      onclick={() => (showCreateDialog = true)}
    >
      <p class="text-muted-foreground text-sm">No organizations yet.</p>
      <p class="text-foreground mt-1 text-sm font-medium">
        Create your first organization to get started
      </p>
    </button>
  {/if}
</div>

<!-- Dialogs -->
<OrganizationCreationDialog bind:open={showCreateDialog} />

{#if deleteTarget}
  <OrganizationDeleteDialog organization={deleteTarget} bind:open={showDeleteDialog} />
{/if}
