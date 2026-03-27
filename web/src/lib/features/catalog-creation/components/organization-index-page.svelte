<script lang="ts">
  import type { Organization } from '$lib/api/contracts'
  import { organizationPath } from '$lib/stores/app-context'
  import { Button } from '$ui/button'
  import OrganizationBulkArchiveBar from './organization-bulk-archive-bar.svelte'
  import OrganizationCreationDialog from './organization-creation-dialog.svelte'
  import OrganizationDeleteDialog from './organization-delete-dialog.svelte'

  let { organizations }: { organizations: Organization[] } = $props()

  let showCreateDialog = $state(false)
  let deleteTarget = $state<Organization | null>(null)
  let showDeleteDialog = $state(false)
  let selectedIds = $state(new Set<string>())

  const allSelected = $derived(
    organizations.length > 0 && selectedIds.size === organizations.length,
  )
  const someSelected = $derived(selectedIds.size > 0 && selectedIds.size < organizations.length)

  function toggleSelect(id: string) {
    const next = new Set(selectedIds)
    if (next.has(id)) next.delete(id)
    else next.add(id)
    selectedIds = next
  }

  function toggleAll() {
    if (allSelected) {
      selectedIds = new Set()
    } else {
      selectedIds = new Set(organizations.map((organization) => organization.id))
    }
  }

  function clearSelection() {
    selectedIds = new Set()
  }

  function openDelete(organization: Organization) {
    deleteTarget = organization
    showDeleteDialog = true
  }
</script>

<div class="mx-auto flex w-full max-w-6xl flex-col gap-8 px-6 py-6">
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

  <OrganizationBulkArchiveBar selectedIds={[...selectedIds]} onClear={clearSelection} />

  {#if organizations.length > 0}
    <div class="border-border divide-border divide-y rounded-lg border">
      <div class="flex items-center gap-4 px-5 py-3">
        <input
          id="orgs-select-all"
          type="checkbox"
          class="size-4 accent-current"
          checked={allSelected}
          indeterminate={someSelected}
          onchange={toggleAll}
        />
        <label for="orgs-select-all" class="text-muted-foreground text-xs font-medium">
          {allSelected ? 'Deselect all' : 'Select all'}
        </label>
      </div>

      {#each organizations as organization (organization.id)}
        <div
          class="flex items-center justify-between gap-4 px-5 py-4 transition-colors {selectedIds.has(
            organization.id,
          )
            ? 'bg-muted/40'
            : ''}"
        >
          <div class="flex min-w-0 items-center gap-4">
            <input
              id={`org-select-${organization.id}`}
              type="checkbox"
              class="size-4 shrink-0 accent-current"
              checked={selectedIds.has(organization.id)}
              aria-label={`Select organization ${organization.name}`}
              onchange={() => toggleSelect(organization.id)}
            />
            <label class="sr-only" for={`org-select-${organization.id}`}>
              Select organization {organization.name}
            </label>
            <div class="min-w-0">
              <a
                href={organizationPath(organization.id)}
                class="text-foreground truncate text-sm font-medium hover:underline"
              >
                {organization.name}
              </a>
              <p class="text-muted-foreground mt-0.5 truncate text-xs">
                {organization.slug}
              </p>
            </div>
          </div>

          <div class="flex shrink-0 items-center gap-2">
            <Button variant="ghost" size="sm" href={organizationPath(organization.id)}>Open</Button>
            <Button
              variant="ghost"
              size="sm"
              class="text-destructive hover:text-destructive"
              onclick={() => openDelete(organization)}
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

<OrganizationCreationDialog bind:open={showCreateDialog} />

{#if deleteTarget}
  <OrganizationDeleteDialog organization={deleteTarget} bind:open={showDeleteDialog} />
{/if}
