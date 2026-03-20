<script lang="ts">
  import { appStore } from '$lib/stores/app.svelte'
  import { getTicketDetail, listStatuses, listWorkflows } from '$lib/api/openase'
  import {
    Sheet,
    SheetContent,
    SheetHeader,
    SheetTitle,
    SheetDescription,
  } from '$ui/sheet'
  import { Tabs, TabsContent, TabsList, TabsTrigger } from '$ui/tabs'
  import { ApiError } from '$lib/api/client'
  import TicketHeader from './ticket-header.svelte'
  import TicketSummary from './ticket-summary.svelte'
  import TicketRepos from './ticket-repos.svelte'
  import TicketHooks from './ticket-hooks.svelte'
  import TicketActivityList from './ticket-activity.svelte'
  import type { TicketDetail, HookExecution, TicketActivity } from '../types'

  let {
    open = $bindable(false),
    projectId,
    ticketId,
    onOpenChange,
  }: {
    open?: boolean
    projectId?: string | null
    ticketId?: string | null
    onOpenChange?: (open: boolean) => void
  } = $props()

  let loading = $state(false)
  let error = $state('')
  let ticket = $state<TicketDetail | null>(null)
  let hooks = $state<HookExecution[]>([])
  let activities = $state<TicketActivity[]>([])

  $effect(() => {
    onOpenChange?.(open)
  })

  $effect(() => {
    const currentProjectId = projectId
    const currentTicketId = ticketId
    if (!open || !currentProjectId || !currentTicketId) {
      if (!open) {
        ticket = null
        hooks = []
        activities = []
        error = ''
      }
      return
    }

    let cancelled = false

    const load = async () => {
      loading = true
      error = ''

      try {
        const [detailPayload, statusPayload, workflowPayload] = await Promise.all([
          getTicketDetail(currentProjectId, currentTicketId),
          listStatuses(currentProjectId),
          listWorkflows(currentProjectId),
        ])

        if (cancelled) return

        const statusMap = new Map(
          statusPayload.statuses.map((status) => [status.id, status]),
        )
        const workflowMap = new Map(
          workflowPayload.workflows.map((workflow) => [workflow.id, workflow]),
        )

        const detailTicket = detailPayload.ticket
        const status = statusMap.get(detailTicket.status_id)
        const workflow = detailTicket.workflow_id
          ? workflowMap.get(detailTicket.workflow_id)
          : null

        ticket = {
          id: detailTicket.id,
          identifier: detailTicket.identifier,
          title: detailTicket.title,
          description: detailTicket.description,
          status: {
            id: detailTicket.status_id,
            name: detailTicket.status_name,
            color: status?.color ?? '#94a3b8',
          },
          priority: normalizePriority(detailTicket.priority),
          type: normalizeType(detailTicket.type),
          workflow: workflow
            ? {
                id: workflow.id,
                name: workflow.name,
                type: workflow.type,
              }
            : undefined,
          repoScopes: detailPayload.repo_scopes.map((scope) => ({
            repoName: scope.repo?.name ?? 'Detached repository',
            branchName: scope.branch_name,
            prUrl: scope.pull_request_url ?? undefined,
            prStatus: scope.pr_status ?? undefined,
            ciStatus: scope.ci_status ?? undefined,
          })),
          attemptCount: detailTicket.attempt_count,
          costAmount: detailTicket.cost_amount,
          budgetUsd: detailTicket.budget_usd,
          dependencies: detailTicket.dependencies.map((dependency) => ({
            id: dependency.id,
            identifier: dependency.target.identifier,
            title: dependency.target.title,
            relation: dependency.type,
          })),
          children: detailTicket.children.map((child) => ({
            id: child.id,
            identifier: child.identifier,
            title: child.title,
            status: child.status_name,
          })),
          createdBy: detailTicket.created_by,
          createdAt: detailTicket.created_at,
          updatedAt: detailTicket.created_at,
        }

        hooks = detailPayload.hook_history.map((entry) => ({
          id: entry.id,
          hookName: entry.event_type,
          status: inferHookStatus(entry.event_type, entry.message),
          output: entry.message,
          timestamp: entry.created_at,
        }))

        activities = detailPayload.activity.map((entry) => ({
          id: entry.id,
          type: normalizeActivityType(entry.event_type),
          message: entry.message,
          timestamp: entry.created_at,
          agentName: agentNameFromMetadata(entry.metadata),
        }))
      } catch (caughtError) {
        if (cancelled) return
        error =
          caughtError instanceof ApiError
            ? caughtError.detail
            : 'Failed to load ticket detail.'
      } finally {
        if (!cancelled) {
          loading = false
        }
      }
    }

    void load()

    return () => {
      cancelled = true
    }
  })

  function handleClose() {
    appStore.closeRightPanel()
  }

  function normalizePriority(priority: string): TicketDetail['priority'] {
    if (priority === 'urgent' || priority === 'high' || priority === 'medium' || priority === 'low') {
      return priority
    }

    return 'medium'
  }

  function normalizeType(type: string): TicketDetail['type'] {
    if (type === 'feature' || type === 'bugfix' || type === 'refactor' || type === 'chore') {
      return type
    }

    return 'feature'
  }

  function inferHookStatus(eventType: string, message: string): HookExecution['status'] {
    const normalized = `${eventType} ${message}`.toLowerCase()
    if (normalized.includes('fail') || normalized.includes('error')) return 'fail'
    if (normalized.includes('running') || normalized.includes('start')) return 'running'
    if (normalized.includes('timeout')) return 'timeout'
    return 'pass'
  }

  function normalizeActivityType(eventType: string) {
    if (eventType === 'status_changed') return 'status_change'
    if (eventType === 'agent_started') return 'started'
    if (eventType === 'agent_completed') return 'completed'
    if (eventType === 'comment_added') return 'comment'
    return eventType
  }

  function agentNameFromMetadata(metadata: Record<string, unknown>) {
    const value = metadata.agent_name
    return typeof value === 'string' ? value : undefined
  }
</script>

<Sheet bind:open>
  <SheetContent side="right" class="w-full sm:max-w-lg p-0 flex flex-col" showCloseButton={false}>
    <SheetHeader class="sr-only">
      <SheetTitle>{ticket?.identifier ?? 'Ticket detail'}</SheetTitle>
      <SheetDescription>Ticket detail drawer</SheetDescription>
    </SheetHeader>

    {#if loading}
      <div class="flex flex-1 items-center justify-center text-sm text-muted-foreground">
        Loading ticket detail…
      </div>
    {:else if error}
      <div class="flex flex-1 items-center justify-center px-6 text-center text-sm text-destructive">
        {error}
      </div>
    {:else if ticket}
      <TicketHeader {ticket} onClose={handleClose} />

      <Tabs value="summary" class="flex flex-1 flex-col overflow-hidden">
        <TabsList class="mx-5 shrink-0">
          <TabsTrigger value="summary">Summary</TabsTrigger>
          <TabsTrigger value="code">Code</TabsTrigger>
          <TabsTrigger value="hooks">Hooks</TabsTrigger>
          <TabsTrigger value="activity">Activity</TabsTrigger>
        </TabsList>

        <div class="flex-1 overflow-y-auto">
          <TabsContent value="summary" class="mt-0">
            <TicketSummary {ticket} />
          </TabsContent>

          <TabsContent value="code" class="mt-0">
            <TicketRepos {ticket} />
          </TabsContent>

          <TabsContent value="hooks" class="mt-0">
            <TicketHooks {hooks} />
          </TabsContent>

          <TabsContent value="activity" class="mt-0">
            <TicketActivityList {activities} />
          </TabsContent>
        </div>
      </Tabs>
    {/if}
  </SheetContent>
</Sheet>
