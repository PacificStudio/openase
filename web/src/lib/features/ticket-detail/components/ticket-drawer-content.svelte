<script lang="ts">
  import { Tabs, TabsContent, TabsList, TabsTrigger } from '$ui/tabs'
  import TicketActivityList from './ticket-activity.svelte'
  import TicketHeader from './ticket-header.svelte'
  import TicketHooks from './ticket-hooks.svelte'
  import TicketRepos from './ticket-repos.svelte'
  import TicketSummary from './ticket-summary.svelte'
  import type {
    HookExecution,
    TicketActivity,
    TicketDetail,
    TicketReferenceOption,
    TicketRepoOption,
    TicketStatusOption,
  } from '../types'

  let {
    ticket,
    hooks,
    activities,
    statuses,
    dependencyCandidates,
    repoOptions,
    mutationError = '',
    mutationNotice = '',
    savingFields = false,
    creatingDependency = false,
    deletingDependencyId = null,
    creatingRepoScope = false,
    updatingRepoScopeId = null,
    deletingRepoScopeId = null,
    onClose,
    onSaveFields,
    onAddDependency,
    onDeleteDependency,
    onCreateScope,
    onUpdateScope,
    onDeleteScope,
  }: {
    ticket: TicketDetail
    hooks: HookExecution[]
    activities: TicketActivity[]
    statuses: TicketStatusOption[]
    dependencyCandidates: TicketReferenceOption[]
    repoOptions: TicketRepoOption[]
    mutationError?: string
    mutationNotice?: string
    savingFields?: boolean
    creatingDependency?: boolean
    deletingDependencyId?: string | null
    creatingRepoScope?: boolean
    updatingRepoScopeId?: string | null
    deletingRepoScopeId?: string | null
    onClose?: () => void
    onSaveFields?: (draft: { title: string; description: string; statusId: string }) => void
    onAddDependency?: (draft: { targetTicketId: string; relation: string }) => void
    onDeleteDependency?: (dependencyId: string) => void
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

<TicketHeader {ticket} {onClose} />

{#if mutationNotice}
  <div class="border-border bg-muted/40 mx-5 mt-4 rounded-md border px-3 py-2 text-xs">
    {mutationNotice}
  </div>
{/if}

{#if mutationError}
  <div
    class="border-destructive/30 bg-destructive/10 text-destructive mx-5 mt-4 rounded-md border px-3 py-2 text-xs"
  >
    {mutationError}
  </div>
{/if}

<Tabs value="summary" class="flex flex-1 flex-col overflow-hidden">
  <TabsList class="mx-5 mt-4 shrink-0">
    <TabsTrigger value="summary">Summary</TabsTrigger>
    <TabsTrigger value="code">Code</TabsTrigger>
    <TabsTrigger value="hooks">Hooks</TabsTrigger>
    <TabsTrigger value="activity">Activity</TabsTrigger>
  </TabsList>

  <div class="flex-1 overflow-y-auto">
    <TabsContent value="summary" class="mt-0">
      <TicketSummary
        {ticket}
        {statuses}
        availableTickets={dependencyCandidates}
        {savingFields}
        {creatingDependency}
        {deletingDependencyId}
        {onSaveFields}
        {onAddDependency}
        {onDeleteDependency}
      />
    </TabsContent>

    <TabsContent value="code" class="mt-0">
      <TicketRepos
        {ticket}
        repos={repoOptions}
        {creatingRepoScope}
        {updatingRepoScopeId}
        {deletingRepoScopeId}
        {onCreateScope}
        {onUpdateScope}
        {onDeleteScope}
      />
    </TabsContent>

    <TabsContent value="hooks" class="mt-0">
      <TicketHooks {hooks} />
    </TabsContent>

    <TabsContent value="activity" class="mt-0">
      <TicketActivityList {activities} />
    </TabsContent>
  </div>
</Tabs>
