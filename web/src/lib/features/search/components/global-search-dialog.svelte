<script lang="ts">
  import { goto } from '$app/navigation'
  import type { Agent, Organization, Project, Ticket, Workflow } from '$lib/api/contracts'
  import { ApiError } from '$lib/api/client'
  import { listAgents, listTickets, listWorkflows } from '$lib/api/openase'
  import type { ProjectSection } from '$lib/stores/app-context'
  import * as Command from '$ui/command'
  import { Badge } from '$ui/badge'
  import { buildSearchIndex, groupSearchItems } from '../model'
  import type { SearchItem, SearchItemAction } from '../types'

  let {
    open = $bindable(false),
    organizations = [],
    projects = [],
    currentOrg = null,
    currentProject = null,
    currentSection = 'dashboard' as ProjectSection,
    newTicketEnabled = false,
    onToggleTheme,
    onNewTicket,
    onOpenTicket,
    onOpenProjectAssistant,
  }: {
    open?: boolean
    organizations?: Organization[]
    projects?: Project[]
    currentOrg?: Organization | null
    currentProject?: Project | null
    currentSection?: ProjectSection
    newTicketEnabled?: boolean
    onToggleTheme?: () => void
    onNewTicket?: () => void
    onOpenTicket?: (ticketId: string) => void
    onOpenProjectAssistant?: (initialPrompt?: string) => void
  } = $props()

  let tickets = $state<Ticket[]>([])
  let workflows = $state<Workflow[]>([])
  let agents = $state<Agent[]>([])
  let loading = $state(false)
  let error = $state('')
  let commandValue = $state('')
  let indexedProjectId = $state<string | null>(null)
  let activeProjectId = $state<string | null>(null)

  const searchItems = $derived(
    buildSearchIndex({
      organizations,
      projects,
      currentOrg,
      currentProject,
      currentSection,
      tickets,
      workflows,
      agents,
      newTicketEnabled,
    }),
  )
  const groupedItems = $derived(groupSearchItems(searchItems))

  $effect(() => {
    const nextProjectId = currentProject?.id ?? null
    if (activeProjectId === nextProjectId) {
      return
    }

    activeProjectId = nextProjectId
    indexedProjectId = null
    tickets = []
    workflows = []
    agents = []
    error = ''
    loading = false
  })

  $effect(() => {
    if (open) {
      return
    }

    commandValue = ''
  })

  $effect(() => {
    const projectId = currentProject?.id ?? null
    if (!open || !projectId || indexedProjectId === projectId) {
      return
    }

    let cancelled = false

    const load = async () => {
      loading = true
      error = ''

      try {
        const [ticketPayload, workflowPayload, agentPayload] = await Promise.all([
          listTickets(projectId),
          listWorkflows(projectId),
          listAgents(projectId),
        ])
        if (cancelled) {
          return
        }

        tickets = ticketPayload.tickets
        workflows = workflowPayload.workflows
        agents = agentPayload.agents
        indexedProjectId = projectId
      } catch (caughtError) {
        if (cancelled) {
          return
        }

        error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to load search data.'
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

  async function handleSelect(item: SearchItem) {
    const selectedQuery = commandValue.trim()
    open = false
    commandValue = ''
    await executeAction(item.action, selectedQuery)
  }

  async function executeAction(action: SearchItemAction, selectedQuery: string) {
    switch (action.kind) {
      case 'navigate':
        await goto(action.href)
        return
      case 'open_ticket':
        onOpenTicket?.(action.ticketId)
        return
      case 'open_project_ai':
        onOpenProjectAssistant?.(selectedQuery || undefined)
        return
      case 'new_ticket':
        onNewTicket?.()
        return
      case 'toggle_theme':
        onToggleTheme?.()
        return
    }
  }
</script>

<Command.Dialog
  bind:open
  bind:value={commandValue}
  title="Global Search"
  description="Search pages, tickets, workflows, agents, projects, and commands."
  class="max-w-3xl border border-white/10 shadow-2xl"
>
  <Command.Input placeholder="Search pages, tickets, workflows, agents, and commands..." />
  <Command.List class="max-h-[26rem]">
    <Command.Empty>
      <div class="text-muted-foreground px-4 py-8 text-center text-sm">
        No search results found.
      </div>
    </Command.Empty>

    {#if loading}
      <div class="text-muted-foreground px-4 py-3 text-sm">Loading project search index…</div>
    {/if}

    {#if error}
      <div class="text-destructive bg-destructive/10 mx-2 mb-2 rounded-lg px-3 py-2 text-sm">
        {error}
      </div>
    {/if}

    {#each groupedItems as group (group.heading)}
      <Command.Group heading={group.heading}>
        {#each group.items as item (item.id)}
          <Command.Item
            value={`${item.kind}:${item.id} ${item.searchText}`}
            onSelect={() => void handleSelect(item)}
          >
            <div class="min-w-0 flex-1">
              <div class="flex items-center gap-2">
                <span class="truncate font-medium">{item.title}</span>
                {#if item.badge}
                  <Badge variant="outline" class="text-[10px] tracking-wide uppercase">
                    {item.badge}
                  </Badge>
                {/if}
              </div>
              <p class="text-muted-foreground mt-0.5 truncate text-xs">{item.subtitle}</p>
            </div>
          </Command.Item>
        {/each}
      </Command.Group>
    {/each}
  </Command.List>
</Command.Dialog>
