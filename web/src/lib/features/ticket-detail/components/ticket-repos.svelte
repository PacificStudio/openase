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
    }) => Promise<boolean> | boolean
    onUpdateScope?: (
      scopeId: string,
      draft: {
        branchName: string
        pullRequestUrl: string
      },
    ) => void
    onDeleteScope?: (scopeId: string) => void
  } = $props()

  let createOpen = $state(false)

  async function handleCreate(draft: {
    repoId: string
    branchName: string
    pullRequestUrl: string
  }) {
    const accepted = (await onCreateScope?.(draft)) ?? false
    if (accepted) {
      createOpen = false
    }
    return accepted
  }
</script>

<section class="space-y-3">
  <div class="flex items-center justify-between gap-3">
    <span class="text-muted-foreground text-[10px] font-medium tracking-wider uppercase">
      Repositories
    </span>

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
          <Dialog.Description
            >Link a repository branch and PR reference to this ticket.</Dialog.Description
          >
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
