<script lang="ts">
  import { buttonVariants } from '$ui/button'
  import * as Dialog from '$ui/dialog'
  import Plus from '@lucide/svelte/icons/plus'
  import TicketRepoCreateForm from './ticket-repo-create-form.svelte'
  import TicketRepoScopeCard from './ticket-repo-scope-card.svelte'
  import type { TicketDetail, TicketRepoOption } from '../types'

  let {
    ticket,
    repos,
    creatingRepoScope = false,
    updatingRepoScopeId = null,
    deletingRepoScopeId = null,
    onCreateScope,
    onUpdateScope,
    onDeleteScope,
  }: {
    ticket: TicketDetail
    repos: TicketRepoOption[]
    creatingRepoScope?: boolean
    updatingRepoScopeId?: string | null
    deletingRepoScopeId?: string | null
    onCreateScope?: (draft: {
      repoId: string
      branchName: string
      pullRequestUrl: string
      prStatus: string
      ciStatus: string
      isPrimaryScope: boolean
    }) => Promise<boolean> | boolean
    onUpdateScope?: (
      scopeId: string,
      draft: {
        branchName: string
        pullRequestUrl: string
        prStatus: string
        ciStatus: string
        isPrimaryScope: boolean
      },
    ) => void
    onDeleteScope?: (scopeId: string) => void
  } = $props()

  let createOpen = $state(false)

  async function handleCreate(draft: {
    repoId: string
    branchName: string
    pullRequestUrl: string
    prStatus: string
    ciStatus: string
    isPrimaryScope: boolean
  }) {
    const accepted = (await onCreateScope?.(draft)) ?? false
    if (accepted) {
      createOpen = false
    }
    return accepted
  }
</script>

<div class="flex flex-col gap-4">
  <section class="space-y-3">
    <div class="flex items-start justify-between gap-3">
      <div>
        <h3 class="text-sm font-medium">Repositories & PRs</h3>
        <p class="text-muted-foreground mt-1 text-xs">
          Review linked repositories at a glance, then open Edit when you need to update branch, PR,
          CI, or primary scope details.
        </p>
      </div>

      <Dialog.Root bind:open={createOpen}>
        <Dialog.Trigger
          class={buttonVariants({ variant: 'outline', size: 'icon-sm' })}
          aria-label="Add repo scope"
          disabled={!repos.length}
        >
          <Plus class="size-3.5" />
        </Dialog.Trigger>
        <Dialog.Content class="sm:max-w-xl">
          <Dialog.Header>
            <Dialog.Title>Add repo scope</Dialog.Title>
            <Dialog.Description>
              Link a repository branch and its PR/CI lifecycle to this ticket.
            </Dialog.Description>
          </Dialog.Header>

          <TicketRepoCreateForm
            {repos}
            creating={creatingRepoScope}
            onCreate={handleCreate}
            onCancel={() => {
              createOpen = false
            }}
          />
        </Dialog.Content>
      </Dialog.Root>
    </div>

    {#if ticket.repoScopes.length === 0}
      <p
        class="text-muted-foreground rounded-md border border-dashed px-3 py-4 text-center text-xs"
      >
        No repositories linked yet.
      </p>
    {/if}

    {#each ticket.repoScopes as scope (scope.id)}
      <TicketRepoScopeCard
        {scope}
        saving={updatingRepoScopeId === scope.id}
        deleting={deletingRepoScopeId === scope.id}
        onSave={onUpdateScope}
        onDelete={onDeleteScope}
      />
    {/each}
  </section>
</div>
