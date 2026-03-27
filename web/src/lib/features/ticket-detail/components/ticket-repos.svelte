<script lang="ts">
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
    }) => void
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
</script>

<div class="flex flex-col gap-4">
  <TicketRepoCreateForm {repos} creating={creatingRepoScope} onCreate={onCreateScope} />

  <section class="space-y-3">
    <div>
      <h3 class="text-sm font-medium">Repositories & PRs</h3>
      <p class="text-muted-foreground mt-1 text-xs">
        Edit branch, PR metadata, CI state, and primary scope assignment inline.
      </p>
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
