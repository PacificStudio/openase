<script lang="ts">
  import { onMount } from 'svelte'
  import {
    Activity,
    ArrowLeft,
    Cable,
    Clock3,
    ExternalLink,
    GitBranch,
    LoaderCircle,
    TriangleAlert,
    Waypoints,
  } from '@lucide/svelte'
  import { connectEventStream, type SSEFrame, type StreamConnectionState } from '$lib/api/sse'
  import { Badge } from '$lib/components/ui/badge'
  import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
  } from '$lib/components/ui/card'

  type Project = {
    id: string
    name: string
    slug: string
    description: string
    status: string
  }

  type TicketReference = {
    id: string
    identifier: string
    title: string
    status_id: string
    status_name: string
  }

  type TicketDependency = {
    id: string
    type: string
    target: TicketReference
  }

  type Ticket = {
    id: string
    project_id: string
    identifier: string
    title: string
    description: string
    status_id: string
    status_name: string
    priority: string
    type: string
    workflow_id?: string | null
    created_by: string
    parent?: TicketReference | null
    children: TicketReference[]
    dependencies: TicketDependency[]
    external_ref: string
    budget_usd: number
    cost_amount: number
    attempt_count: number
    consecutive_errors: number
    next_retry_at?: string | null
    retry_paused: boolean
    pause_reason: string
    created_at: string
  }

  type ProjectRepo = {
    id: string
    project_id: string
    name: string
    repository_url: string
    default_branch: string
    clone_path?: string | null
    is_primary: boolean
    labels: string[]
  }

  type TicketRepoScope = {
    id: string
    ticket_id: string
    repo_id: string
    repo?: ProjectRepo | null
    branch_name: string
    pull_request_url?: string | null
    pr_status: string
    ci_status: string
    is_primary_scope: boolean
  }

  type ActivityEvent = {
    id: string
    project_id: string
    ticket_id?: string | null
    agent_id?: string | null
    event_type: string
    message: string
    metadata: Record<string, unknown>
    created_at: string
  }

  type StreamEnvelope = {
    topic: string
    type: string
    payload?: unknown
    published_at: string
  }

  type TicketDetailPayload = {
    ticket: Ticket
    repo_scopes: TicketRepoScope[]
    activity: ActivityEvent[]
    hook_history: ActivityEvent[]
  }

  const panelClass = 'border-border/80 bg-background/85 backdrop-blur'

  let projectId = $state('')
  let ticketId = $state('')
  let loading = $state(true)
  let refreshing = $state(false)
  let errorMessage = $state('')
  let project = $state<Project | null>(null)
  let detail = $state<TicketDetailPayload | null>(null)
  let ticketStreamState = $state<StreamConnectionState>('idle')
  let activityStreamState = $state<StreamConnectionState>('idle')
  let hookStreamState = $state<StreamConnectionState>('idle')
  let loadInFlight = false
  let reloadQueued = false

  onMount(() => {
    const params = new URLSearchParams(window.location.search)
    projectId = params.get('project') ?? ''
    ticketId = params.get('id') ?? ''
    void loadDetail()
    if (!projectId || !ticketId) {
      return
    }

    const closeTicketStream = connectEventStream(`/api/v1/projects/${projectId}/tickets/stream`, {
      onEvent: handleTicketFrame,
      onStateChange: (state) => {
        ticketStreamState = state
      },
      onError: (error) => {
        errorMessage = readErrorMessage(error)
      },
    })
    const closeActivityStream = connectEventStream(
      `/api/v1/projects/${projectId}/activity/stream`,
      {
        onEvent: handleActivityFrame,
        onStateChange: (state) => {
          activityStreamState = state
        },
        onError: (error) => {
          errorMessage = readErrorMessage(error)
        },
      },
    )
    const closeHookStream = connectEventStream(`/api/v1/projects/${projectId}/hooks/stream`, {
      onEvent: handleHookFrame,
      onStateChange: (state) => {
        hookStreamState = state
      },
      onError: (error) => {
        errorMessage = readErrorMessage(error)
      },
    })

    return () => {
      closeTicketStream()
      closeActivityStream()
      closeHookStream()
    }
  })

  async function loadDetail(options: { silent?: boolean } = {}) {
    if (loadInFlight) {
      reloadQueued = true
      return
    }

    if (!projectId || !ticketId) {
      loading = false
      errorMessage = 'Missing project or ticket identifier in the URL.'
      return
    }

    loadInFlight = true
    if (options.silent) {
      refreshing = true
    } else {
      loading = true
    }
    errorMessage = ''

    try {
      const [detailPayload, projectPayload] = await Promise.all([
        api<TicketDetailPayload>(`/api/v1/projects/${projectId}/tickets/${ticketId}/detail`),
        api<{ project: Project }>(`/api/v1/projects/${projectId}`),
      ])
      detail = detailPayload
      project = projectPayload.project
    } catch (error) {
      errorMessage = readErrorMessage(error)
    } finally {
      loadInFlight = false
      loading = false
      refreshing = false
      if (reloadQueued) {
        reloadQueued = false
        void loadDetail({ silent: true })
      }
    }
  }

  function handleTicketFrame(frame: SSEFrame) {
    const envelope = parseStreamEnvelope(frame)
    if (!envelope) {
      return
    }
    if (extractRelatedTicketID(envelope.payload) !== ticketId) {
      return
    }
    void loadDetail({ silent: true })
  }

  function handleActivityFrame(frame: SSEFrame) {
    const envelope = parseStreamEnvelope(frame)
    if (!envelope) {
      return
    }
    const item = parseActivityEvent(envelope.payload, envelope.published_at)
    if (!item || item.ticket_id !== ticketId || !detail) {
      return
    }

    detail = {
      ...detail,
      activity: dedupeEvents([item, ...detail.activity]).slice(0, 100),
      hook_history: isHookEvent(item)
        ? dedupeEvents([item, ...detail.hook_history]).slice(0, 50)
        : detail.hook_history,
    }
  }

  function handleHookFrame(frame: SSEFrame) {
    const envelope = parseStreamEnvelope(frame)
    if (!envelope) {
      return
    }
    if (extractRelatedTicketID(envelope.payload) !== ticketId) {
      return
    }
    void loadDetail({ silent: true })
  }

  function parseStreamEnvelope(frame: SSEFrame): StreamEnvelope | null {
    try {
      const parsed = JSON.parse(frame.data)
      if (!isRecord(parsed)) {
        return null
      }
      const topic = readString(parsed, 'topic')
      const type = readString(parsed, 'type')
      const publishedAt = readString(parsed, 'published_at')
      if (!topic || !type || !publishedAt) {
        return null
      }
      return { topic, type, payload: parsed.payload, published_at: publishedAt }
    } catch {
      return null
    }
  }

  function parseActivityEvent(payload: unknown, publishedAt: string): ActivityEvent | null {
    const source = readRecord(payload)
    if (!source) {
      return null
    }
    const id = readString(source, 'id')
    const projectID = readString(source, 'project_id')
    const eventType = readString(source, 'event_type') ?? readString(source, 'type')
    if (!id || !projectID || !eventType) {
      return null
    }
    return {
      id,
      project_id: projectID,
      ticket_id: readNullableString(source, 'ticket_id'),
      agent_id: readNullableString(source, 'agent_id'),
      event_type: eventType,
      message: readString(source, 'message') ?? '',
      metadata: readRecord(source.metadata) ?? {},
      created_at: readString(source, 'created_at') ?? publishedAt,
    }
  }

  function extractRelatedTicketID(payload: unknown): string | null {
    const source = readRecord(payload)
    if (!source) {
      return null
    }
    const nestedTicket = readRecord(source.ticket)
    if (nestedTicket) {
      return readString(nestedTicket, 'id') ?? readNullableString(nestedTicket, 'ticket_id') ?? null
    }
    return readNullableString(source, 'ticket_id') ?? readString(source, 'ticketId') ?? null
  }

  function dedupeEvents(items: ActivityEvent[]) {
    const seen = new Set<string>()
    return items.filter((item) => {
      if (seen.has(item.id)) {
        return false
      }
      seen.add(item.id)
      return true
    })
  }

  function isHookEvent(item: ActivityEvent) {
    if (item.event_type.toLowerCase().includes('hook')) {
      return true
    }
    return ['hook', 'hook_name', 'hook_stage', 'hook_result', 'hook_outcome'].some(
      (key) => key in item.metadata,
    )
  }

  function isRecord(value: unknown): value is Record<string, unknown> {
    return typeof value === 'object' && value !== null && !Array.isArray(value)
  }

  function readRecord(value: unknown) {
    return isRecord(value) ? value : null
  }

  function readString(source: Record<string, unknown>, key: string) {
    const value = source[key]
    return typeof value === 'string' && value.trim() ? value : undefined
  }

  function readNullableString(source: Record<string, unknown>, key: string) {
    const value = source[key]
    if (value === null) {
      return null
    }
    return typeof value === 'string' ? value : undefined
  }

  async function api<T>(path: string): Promise<T> {
    const response = await fetch(path)
    const payload = await response.json().catch(() => ({}))
    if (!response.ok) {
      throw new Error(
        readString(payload, 'message') ??
          readString(payload, 'error') ??
          `request failed with status ${response.status}`,
      )
    }
    return payload as T
  }

  function readErrorMessage(error: unknown) {
    return error instanceof Error ? error.message : 'Request failed'
  }

  function formatTimestamp(value: string) {
    const parsed = Date.parse(value)
    if (Number.isNaN(parsed)) {
      return value
    }
    return new Intl.DateTimeFormat(undefined, {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    }).format(parsed)
  }

  function ticketPriorityBadgeClass(priority: string) {
    switch (priority) {
      case 'urgent':
        return 'border-rose-500/25 bg-rose-500/10 text-rose-700'
      case 'high':
        return 'border-amber-500/25 bg-amber-500/10 text-amber-700'
      case 'medium':
        return 'border-sky-500/25 bg-sky-500/10 text-sky-700'
      default:
        return 'border-border/80 bg-background text-muted-foreground'
    }
  }

  function statusBadgeClass(status: string) {
    switch (status) {
      case 'merged':
      case 'passing':
      case 'done':
        return 'border-emerald-500/25 bg-emerald-500/10 text-emerald-700'
      case 'open':
      case 'pending':
        return 'border-sky-500/25 bg-sky-500/10 text-sky-700'
      case 'changes_requested':
      case 'failing':
        return 'border-amber-500/25 bg-amber-500/10 text-amber-700'
      case 'closed':
      case 'failed':
        return 'border-rose-500/25 bg-rose-500/10 text-rose-700'
      default:
        return 'border-border/80 bg-background text-muted-foreground'
    }
  }

  function streamBadgeClass(state: StreamConnectionState) {
    switch (state) {
      case 'live':
        return 'border-emerald-500/25 bg-emerald-500/10 text-emerald-700'
      case 'connecting':
        return 'border-sky-500/25 bg-sky-500/10 text-sky-700'
      case 'retrying':
        return 'border-amber-500/25 bg-amber-500/10 text-amber-700'
      default:
        return 'border-border/80 bg-background text-muted-foreground'
    }
  }
</script>

<svelte:head>
  <title>{detail ? `${detail.ticket.identifier} · Ticket Detail` : 'Ticket Detail'} · OpenASE</title
  >
</svelte:head>

<div
  class="min-h-screen bg-[radial-gradient(circle_at_top_left,_rgba(16,185,129,0.14),_transparent_34%),radial-gradient(circle_at_top_right,_rgba(14,165,233,0.12),_transparent_30%),linear-gradient(180deg,_rgba(248,250,252,0.98),_rgba(241,245,249,0.96))]"
>
  <div class="mx-auto flex w-full max-w-7xl flex-col gap-6 px-4 py-8 sm:px-6 lg:px-8">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <a
        href="/"
        class="border-border/70 bg-background/85 text-foreground hover:border-foreground/20 inline-flex items-center gap-2 rounded-full border px-4 py-2 text-sm font-medium transition"
      >
        <ArrowLeft class="size-4" />
        Back to board
      </a>
      <div class="flex flex-wrap items-center gap-2">
        <span
          class={`inline-flex items-center rounded-full border px-3 py-1 text-xs font-medium ${streamBadgeClass(ticketStreamState)}`}
        >
          <Cable class="mr-1.5 size-3.5" />
          Ticket {ticketStreamState}
        </span>
        <span
          class={`inline-flex items-center rounded-full border px-3 py-1 text-xs font-medium ${streamBadgeClass(activityStreamState)}`}
        >
          <Activity class="mr-1.5 size-3.5" />
          Activity {activityStreamState}
        </span>
        <span
          class={`inline-flex items-center rounded-full border px-3 py-1 text-xs font-medium ${streamBadgeClass(hookStreamState)}`}
        >
          <Waypoints class="mr-1.5 size-3.5" />
          Hooks {hookStreamState}
        </span>
      </div>
    </div>

    {#if errorMessage}
      <div
        class="border-destructive/25 bg-destructive/10 text-destructive rounded-3xl border px-4 py-3 text-sm"
      >
        {errorMessage}
      </div>
    {/if}

    {#if loading && !detail}
      <div
        class="border-border/70 bg-background/80 flex min-h-[24rem] items-center justify-center rounded-[2rem] border"
      >
        <div class="text-muted-foreground flex items-center gap-3 text-sm">
          <LoaderCircle class="size-4 animate-spin" />
          <span>Loading ticket detail…</span>
        </div>
      </div>
    {:else if !projectId || !ticketId}
      <div
        class="border-border/80 bg-background/80 text-muted-foreground rounded-[2rem] border border-dashed px-6 py-8 text-sm"
      >
        Ticket detail URLs require both `project` and `id` query parameters.
      </div>
    {:else if detail}
      <div class="grid gap-6 xl:grid-cols-[minmax(0,1.4fr)_24rem]">
        <div class="space-y-6">
          <Card class={panelClass}>
            <CardHeader class="border-border/60 gap-4 border-b pb-6">
              <div class="flex flex-wrap items-start justify-between gap-4">
                <div class="space-y-3">
                  <div class="flex flex-wrap items-center gap-2">
                    <Badge variant="outline">{detail.ticket.identifier}</Badge>
                    <Badge variant="outline">{project?.name ?? projectId}</Badge>
                    <span
                      class={`inline-flex rounded-full border px-2.5 py-1 text-[11px] font-medium ${ticketPriorityBadgeClass(detail.ticket.priority)}`}
                    >
                      {detail.ticket.priority}
                    </span>
                  </div>
                  <div>
                    <CardTitle class="text-foreground text-3xl tracking-[-0.04em]"
                      >{detail.ticket.title}</CardTitle
                    >
                    <CardDescription class="text-muted-foreground mt-2 max-w-3xl text-sm leading-6">
                      {detail.ticket.description || 'No description yet.'}
                    </CardDescription>
                  </div>
                </div>
                {#if refreshing}
                  <div
                    class="border-border/70 bg-background/80 text-muted-foreground inline-flex items-center gap-2 rounded-full border px-3 py-1 text-xs"
                  >
                    <LoaderCircle class="size-3 animate-spin" />
                    Refreshing
                  </div>
                {/if}
              </div>
            </CardHeader>
            <CardContent class="grid gap-4 pt-6 md:grid-cols-2 xl:grid-cols-4">
              <div class="border-border/70 bg-background/70 rounded-3xl border p-4">
                <p class="text-muted-foreground text-xs tracking-[0.2em] uppercase">Status</p>
                <p class="text-foreground mt-3 text-lg font-semibold">
                  {detail.ticket.status_name}
                </p>
                <p class="text-muted-foreground mt-2 text-sm">{detail.ticket.type}</p>
              </div>
              <div class="border-border/70 bg-background/70 rounded-3xl border p-4">
                <p class="text-muted-foreground text-xs tracking-[0.2em] uppercase">Linked Repos</p>
                <p class="text-foreground mt-3 text-lg font-semibold">
                  {detail.repo_scopes.length}
                </p>
                <p class="text-muted-foreground mt-2 text-sm">
                  {detail.hook_history.length} hook events captured
                </p>
              </div>
              <div class="border-border/70 bg-background/70 rounded-3xl border p-4">
                <p class="text-muted-foreground text-xs tracking-[0.2em] uppercase">Attempts</p>
                <p class="text-foreground mt-3 text-lg font-semibold">
                  {detail.ticket.attempt_count}
                </p>
                <p class="text-muted-foreground mt-2 text-sm">
                  {detail.ticket.retry_paused
                    ? detail.ticket.pause_reason || 'Retry paused'
                    : `${detail.ticket.consecutive_errors} consecutive errors`}
                </p>
              </div>
              <div class="border-border/70 bg-background/70 rounded-3xl border p-4">
                <p class="text-muted-foreground text-xs tracking-[0.2em] uppercase">Created</p>
                <p class="text-foreground mt-3 text-lg font-semibold">
                  {formatTimestamp(detail.ticket.created_at)}
                </p>
                <p class="text-muted-foreground mt-2 text-sm">{detail.ticket.created_by}</p>
              </div>
            </CardContent>
          </Card>

          <Card class={panelClass}>
            <CardHeader>
              <CardTitle class="flex items-center gap-2">
                <GitBranch class="size-4" />
                <span>Multi-repo PRs</span>
              </CardTitle>
              <CardDescription
                >Each repo scope tracks branch, PR URL, review status, and CI health for this
                ticket.</CardDescription
              >
            </CardHeader>
            <CardContent class="space-y-3">
              {#if detail.repo_scopes.length === 0}
                <div
                  class="border-border/80 bg-muted/30 text-muted-foreground rounded-3xl border border-dashed px-4 py-6 text-sm"
                >
                  No repo scopes are linked to this ticket yet.
                </div>
              {:else}
                {#each detail.repo_scopes as scope}
                  <div class="border-border/70 bg-background/75 rounded-3xl border p-4">
                    <div class="flex flex-wrap items-start justify-between gap-3">
                      <div class="space-y-2">
                        <div class="flex flex-wrap items-center gap-2">
                          <p class="text-foreground text-sm font-semibold">
                            {scope.repo?.name ?? scope.repo_id}
                          </p>
                          {#if scope.is_primary_scope}
                            <Badge variant="secondary">primary</Badge>
                          {/if}
                          <span
                            class={`inline-flex rounded-full border px-2.5 py-1 text-[11px] font-medium ${statusBadgeClass(scope.pr_status)}`}
                          >
                            PR {scope.pr_status}
                          </span>
                          <span
                            class={`inline-flex rounded-full border px-2.5 py-1 text-[11px] font-medium ${statusBadgeClass(scope.ci_status)}`}
                          >
                            CI {scope.ci_status}
                          </span>
                        </div>
                        <p class="text-muted-foreground text-xs">
                          Branch `{scope.branch_name}` on {scope.repo?.default_branch ??
                            'unknown default branch'}
                        </p>
                      </div>
                      {#if scope.pull_request_url}
                        <a
                          href={scope.pull_request_url}
                          target="_blank"
                          rel="noreferrer"
                          class="border-border/70 bg-background text-foreground hover:border-foreground/20 inline-flex items-center gap-2 rounded-full border px-3 py-1.5 text-xs font-medium transition"
                        >
                          <ExternalLink class="size-3.5" />
                          Open PR
                        </a>
                      {:else}
                        <span class="text-muted-foreground text-xs">No PR URL yet</span>
                      {/if}
                    </div>
                    {#if scope.repo?.repository_url}
                      <p class="text-muted-foreground mt-3 text-xs">{scope.repo.repository_url}</p>
                    {/if}
                  </div>
                {/each}
              {/if}
            </CardContent>
          </Card>
        </div>

        <div class="space-y-6">
          <Card class={panelClass}>
            <CardHeader>
              <CardTitle class="flex items-center gap-2">
                <Activity class="size-4" />
                <span>Activity Log</span>
              </CardTitle>
              <CardDescription
                >Newest events land first, and SSE appends new lines automatically.</CardDescription
              >
            </CardHeader>
            <CardContent class="space-y-3">
              {#if detail.activity.length === 0}
                <div
                  class="border-border/80 bg-muted/30 text-muted-foreground rounded-3xl border border-dashed px-4 py-6 text-sm"
                >
                  No ticket activity has been recorded yet.
                </div>
              {:else}
                {#each detail.activity as item}
                  <div class="border-border/70 bg-background/75 rounded-3xl border p-4">
                    <div class="flex items-center justify-between gap-3">
                      <p
                        class="text-muted-foreground text-xs font-semibold tracking-[0.18em] uppercase"
                      >
                        {item.event_type}
                      </p>
                      <span class="text-muted-foreground inline-flex items-center gap-1 text-xs">
                        <Clock3 class="size-3" />
                        {formatTimestamp(item.created_at)}
                      </span>
                    </div>
                    <p class="text-foreground mt-3 text-sm leading-6">
                      {item.message || 'No message payload.'}
                    </p>
                    {#if Object.keys(item.metadata).length > 0}
                      <pre
                        class="border-border/70 bg-muted/25 text-muted-foreground mt-3 overflow-x-auto rounded-2xl border p-3 text-[11px]">{JSON.stringify(
                          item.metadata,
                          null,
                          2,
                        )}</pre>
                    {/if}
                  </div>
                {/each}
              {/if}
            </CardContent>
          </Card>

          <Card class={panelClass}>
            <CardHeader>
              <CardTitle class="flex items-center gap-2">
                <Waypoints class="size-4" />
                <span>Hook History</span>
              </CardTitle>
              <CardDescription
                >Hook-tagged events are broken out so failures are visible without scanning the full
                log.</CardDescription
              >
            </CardHeader>
            <CardContent class="space-y-3">
              {#if detail.hook_history.length === 0}
                <div
                  class="border-border/80 bg-muted/30 text-muted-foreground rounded-3xl border border-dashed px-4 py-6 text-sm"
                >
                  No hook events have been recorded for this ticket yet.
                </div>
              {:else}
                {#each detail.hook_history as item}
                  <div class="border-border/70 bg-background/75 rounded-3xl border p-4">
                    <div class="flex items-center justify-between gap-3">
                      <p
                        class="text-muted-foreground text-xs font-semibold tracking-[0.18em] uppercase"
                      >
                        {item.event_type}
                      </p>
                      <span class="text-muted-foreground inline-flex items-center gap-1 text-xs">
                        <Clock3 class="size-3" />
                        {formatTimestamp(item.created_at)}
                      </span>
                    </div>
                    <p class="text-foreground mt-3 text-sm leading-6">
                      {item.message || 'No message payload.'}
                    </p>
                    {#if Object.keys(item.metadata).length > 0}
                      <pre
                        class="border-border/70 bg-muted/25 text-muted-foreground mt-3 overflow-x-auto rounded-2xl border p-3 text-[11px]">{JSON.stringify(
                          item.metadata,
                          null,
                          2,
                        )}</pre>
                    {/if}
                  </div>
                {/each}
              {/if}
            </CardContent>
          </Card>

          <Card class={panelClass}>
            <CardHeader>
              <CardTitle class="flex items-center gap-2">
                <TriangleAlert class="size-4" />
                <span>Ticket Graph</span>
              </CardTitle>
              <CardDescription
                >Parent links and dependencies stay visible without leaving the detail page.</CardDescription
              >
            </CardHeader>
            <CardContent class="text-muted-foreground space-y-3 text-sm">
              <div class="border-border/70 bg-background/75 rounded-3xl border p-4">
                <p class="text-muted-foreground text-xs tracking-[0.18em] uppercase">Parent</p>
                <p class="text-foreground mt-2 text-sm">
                  {detail.ticket.parent
                    ? `${detail.ticket.parent.identifier} · ${detail.ticket.parent.title}`
                    : 'No parent ticket'}
                </p>
              </div>
              <div class="border-border/70 bg-background/75 rounded-3xl border p-4">
                <p class="text-muted-foreground text-xs tracking-[0.18em] uppercase">Children</p>
                {#if detail.ticket.children.length === 0}
                  <p class="mt-2">No child tickets linked yet.</p>
                {:else}
                  <div class="mt-2 flex flex-wrap gap-2">
                    {#each detail.ticket.children as child}
                      <Badge variant="outline">{child.identifier}</Badge>
                    {/each}
                  </div>
                {/if}
              </div>
              <div class="border-border/70 bg-background/75 rounded-3xl border p-4">
                <p class="text-muted-foreground text-xs tracking-[0.18em] uppercase">
                  Dependencies
                </p>
                {#if detail.ticket.dependencies.length === 0}
                  <p class="mt-2">No explicit dependencies.</p>
                {:else}
                  <ul class="mt-2 space-y-2">
                    {#each detail.ticket.dependencies as dependency}
                      <li
                        class="border-border/70 bg-background text-foreground rounded-2xl border px-3 py-2"
                      >
                        {dependency.type} · {dependency.target.identifier} · {dependency.target
                          .title}
                      </li>
                    {/each}
                  </ul>
                {/if}
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    {/if}
  </div>
</div>
