<script lang="ts">
  import { untrack } from 'svelte'
  import { goto } from '$app/navigation'
  import { ApiError } from '$lib/api/client'
  import { createTicket, listProjectRepos } from '$lib/api/openase'
  import type { ProjectRepoRecord, TicketStatus } from '$lib/api/contracts'
  import { appStore } from '$lib/stores/app.svelte'
  import { projectPath } from '$lib/stores/app-context'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Textarea } from '$ui/textarea'
  import * as Select from '$ui/select'
  import { Loader2, Ticket, Info, CheckCircle2 } from '@lucide/svelte'
  import { getBootstrapPreset } from '../model'

  let {
    projectId,
    orgId,
    projectStatus,
    statuses,
    ticketCount,
    onComplete,
  }: {
    projectId: string
    orgId: string
    projectStatus: string
    statuses: TicketStatus[]
    ticketCount: number
    onComplete: () => void
  } = $props()

  const preset = $derived(getBootstrapPreset(projectStatus))

  let title = $state(untrack(() => preset.exampleTicketTitle))
  let description = $state('')
  let creating = $state(false)
  let repos = $state<ProjectRepoRecord[]>([])
  let selectedRepoId = $state('')
  let loadedRepos = $state(false)

  const hasTickets = $derived(ticketCount > 0)
  const pickupStatus = $derived(
    statuses.find(
      (status) => status.name.trim().toLowerCase() === preset.pickupStatusName.trim().toLowerCase(),
    ),
  )

  $effect(() => {
    if (loadedRepos) return
    const load = async () => {
      try {
        const payload = await listProjectRepos(projectId)
        repos = payload.repos
        if (payload.repos.length === 1) {
          selectedRepoId = payload.repos[0]!.id
        }
        loadedRepos = true
      } catch {
        // ignore
      }
    }
    void load()
  })

  const pickupStatusLabel = $derived(pickupStatus ? `enter "${pickupStatus.name}"` : '—')

  async function handleCreate() {
    if (!title.trim()) {
      toastStore.error('Enter a ticket title.')
      return
    }
    if (!pickupStatus) {
      toastStore.error(
        'Could not find the recommended pickup status. Check the project status configuration first.',
      )
      return
    }
    creating = true
    try {
      const payload = await createTicket(projectId, {
        title: title.trim(),
        description: description.trim() || undefined,
        status_id: pickupStatus.id,
        repo_scopes: selectedRepoId ? [{ repo_id: selectedRepoId }] : undefined,
      })

      toastStore.success(`Ticket ${payload.ticket.identifier} created.`)
      onComplete()

      // Navigate to ticket detail by opening right panel
      appStore.openRightPanel({ type: 'ticket', id: payload.ticket.id })
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to create the ticket.',
      )
    } finally {
      creating = false
    }
  }
</script>

<div class="space-y-4">
  {#if hasTickets}
    <div
      class="flex items-center gap-3 rounded-lg border border-emerald-200 bg-emerald-50 px-4 py-3 dark:border-emerald-900/50 dark:bg-emerald-950/30"
    >
      <CheckCircle2 class="size-4 shrink-0 text-emerald-600 dark:text-emerald-400" />
      <Ticket class="text-muted-foreground size-4 shrink-0" />
      <div>
        <p class="text-foreground text-sm font-medium">
          {ticketCount} ticket{ticketCount === 1 ? '' : 's'} created
        </p>
        <p class="text-muted-foreground text-xs">Open the Tickets page to view details</p>
      </div>
      <Button
        variant="outline"
        size="sm"
        class="ml-auto"
        onclick={() => void goto(projectPath(orgId, projectId, 'tickets'))}
      >
        View Tickets
      </Button>
    </div>
  {:else}
    <div class="space-y-3">
      <div>
        <p class="text-foreground mb-1 text-xs font-medium">Ticket title</p>
        <Input
          bind:value={title}
          placeholder={preset.exampleTicketTitle}
          class="text-sm"
          autofocus
        />
      </div>

      <div>
        <p class="text-foreground mb-1 text-xs font-medium">Description (optional)</p>
        <Textarea
          bind:value={description}
          placeholder="Describe the task..."
          rows={2}
          class="text-sm"
        />
      </div>

      <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
        {#if repos.length > 1}
          <div>
            <p class="text-foreground mb-1 text-xs font-medium">Repository scope</p>
            <Select.Root
              type="single"
              value={selectedRepoId}
              onValueChange={(v) => {
                if (v) selectedRepoId = v
              }}
            >
              <Select.Trigger class="h-9 w-full text-sm">
                {repos.find((r) => r.id === selectedRepoId)?.name ?? 'Select a repository'}
              </Select.Trigger>
              <Select.Content>
                {#each repos as repo (repo.id)}
                  <Select.Item value={repo.id}>{repo.name}</Select.Item>
                {/each}
              </Select.Content>
            </Select.Root>
          </div>
        {/if}
      </div>

      <!-- Info about what happens next -->
      <div class="bg-muted/50 flex items-start gap-2 rounded-md p-3">
        <Info class="text-muted-foreground mt-0.5 size-3.5 shrink-0" />
        <div class="text-muted-foreground space-y-1 text-xs">
          <p>
            The ticket will {pickupStatusLabel}, and the orchestrator will automatically pick it up
            and assign it to an agent based on the status pickup rules.
          </p>
          <p>
            The agent's progress will appear in real time on the timeline in the ticket details
            view.
          </p>
        </div>
      </div>

      <Button class="w-full" onclick={handleCreate} disabled={creating || !title.trim()}>
        {#if creating}
          <Loader2 class="mr-1.5 size-3.5 animate-spin" />
          Creating...
        {:else}
          <Ticket class="mr-1.5 size-3.5" />
          Create ticket
        {/if}
      </Button>
    </div>
  {/if}
</div>
