<script lang="ts">
  import { invalidateAll } from '$app/navigation'
  import { ApiError } from '$lib/api/client'
  import { deleteOrganization } from '$lib/api/openase'
  import OrganizationCreationDialog from '$lib/features/catalog-creation/components/organization-creation-dialog.svelte'
  import OrganizationDeleteDialog from '$lib/features/catalog-creation/components/organization-delete-dialog.svelte'
  import { organizationPath } from '$lib/stores/app-context'
  import { Button } from '$ui/button'
  import Trash2 from '@lucide/svelte/icons/trash-2'
  import type { PageData } from './$types'

  let { data }: { data: PageData } = $props()

  type OrganizationItem = PageData['organizations'][number]

  const organizations = $derived(data.organizations)

  let showCreateDialog = $state(false)
  let deleteTarget = $state<OrganizationItem | null>(null)
  let showDeleteDialog = $state(false)

  // Multi-select state
  let selectedIds = $state(new Set<string>())
  let bulkDeleting = $state(false)
  let bulkError = $state('')

  const selectedCount = $derived(selectedIds.size)
  const allSelected = $derived(organizations.length > 0 && selectedIds.size === organizations.length)
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
      selectedIds = new Set(organizations.map((o) => o.id))
    }
  }

  function clearSelection() {
    selectedIds = new Set()
  }

  function openDelete(org: OrganizationItem) {
    deleteTarget = org
    showDeleteDialog = true
  }

  async function bulkDelete() {
    if (selectedCount === 0) return
    bulkDeleting = true
    bulkError = ''
    try {
      const results = await Promise.allSettled(
        [...selectedIds].map((id) => deleteOrganization(id)),
      )
      const failures = results.filter((r) => r.status === 'rejected')
      if (failures.length > 0) {
        const first = failures[0] as PromiseRejectedResult
        bulkError =
          first.reason instanceof ApiError
            ? `${failures.length} failed: ${first.reason.detail}`
            : `${failures.length} deletion(s) failed.`
      }
      selectedIds = new Set()
      await invalidateAll()
    } catch {
      bulkError = 'Bulk delete failed.'
    } finally {
      bulkDeleting = false
    }
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
      </p>
    </div>

    <Button onclick={() => (showCreateDialog = true)}>New organization</Button>
  </div>

  <!-- Bulk action bar -->
  {#if selectedCount > 0}
    <div
      class="bg-muted/60 border-border flex items-center justify-between rounded-lg border px-4 py-3"
    >
      <span class="text-foreground text-sm font-medium">
        {selectedCount} selected
      </span>
      <div class="flex items-center gap-2">
        <Button variant="ghost" size="sm" onclick={clearSelection}>Cancel</Button>
        <Button
          variant="destructive"
          size="sm"
          disabled={bulkDeleting}
          onclick={bulkDelete}
        >
          <Trash2 class="mr-1.5 size-4" />
          {bulkDeleting ? 'Deleting...' : `Delete ${selectedCount}`}
        </Button>
      </div>
    </div>
  {/if}

  {#if bulkError}
    <p class="text-destructive text-sm">{bulkError}</p>
  {/if}

  <!-- Org list -->
  {#if organizations.length > 0}
    <div class="border-border divide-border divide-y rounded-lg border">
      <!-- Select-all header -->
      <div class="flex items-center gap-4 px-5 py-3">
        <input
          type="checkbox"
          class="size-4 accent-current"
          checked={allSelected}
          indeterminate={someSelected}
          onchange={toggleAll}
        />
        <span class="text-muted-foreground text-xs font-medium">
          {allSelected ? 'Deselect all' : 'Select all'}
        </span>
      </div>

      {#each organizations as org (org.id)}
        <div
          class="flex items-center justify-between gap-4 px-5 py-4 transition-colors {selectedIds.has(org.id) ? 'bg-muted/40' : ''}"
        >
          <div class="flex min-w-0 items-center gap-4">
            <input
              type="checkbox"
              class="size-4 shrink-0 accent-current"
              checked={selectedIds.has(org.id)}
              onchange={() => toggleSelect(org.id)}
            />
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
          </div>

          <div class="flex shrink-0 items-center gap-2">
            <Button variant="ghost" size="sm" href={organizationPath(org.id)}>Open</Button>
            <Button
              variant="ghost"
              size="sm"
              class="text-destructive hover:text-destructive"
              onclick={() => openDelete(org)}
            >
              Delete
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
