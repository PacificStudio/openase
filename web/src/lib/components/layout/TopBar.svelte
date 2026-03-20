<script lang="ts">
  import { FolderKanban, Menu, PanelRightOpen, RadioTower, Ticket, Workflow } from '@lucide/svelte'
  import { Badge } from '$lib/components/ui/badge'
  import { Button } from '$lib/components/ui/button'
  import type { Organization, Project } from '$lib/types/workspace-shell'

  let {
    selectedOrg = null,
    selectedProject = null,
    pageTitle = 'Overview',
    pageLabel = 'Project workspace',
    pageStatus = 'idle',
    runningAgentCount = 0,
    ticketCount = 0,
    workflowCount = 0,
    statusMessage = '',
    onToggleSidebar,
    onToggleDrawer,
  }: {
    selectedOrg?: Organization | null
    selectedProject?: Project | null
    pageTitle?: string
    pageLabel?: string
    pageStatus?: string
    runningAgentCount?: number
    ticketCount?: number
    workflowCount?: number
    statusMessage?: string
    onToggleSidebar?: () => void
    onToggleDrawer?: () => void
  } = $props()
</script>

<div class="border-border/70 bg-background/86 rounded-[1.6rem] border px-4 py-3 shadow-sm backdrop-blur">
  <div class="flex min-h-14 flex-wrap items-center justify-between gap-3">
    <div class="flex min-w-0 items-center gap-2 sm:gap-3">
      <Button type="button" variant="outline" size="icon" class="xl:hidden" onclick={onToggleSidebar}>
        <Menu class="size-4" />
      </Button>

      <a
        href="/"
        class="bg-foreground text-background inline-flex size-10 shrink-0 items-center justify-center rounded-2xl text-sm font-semibold tracking-[0.24em]"
      >
        OA
      </a>

      <div class="min-w-0">
        <div class="flex min-w-0 flex-wrap items-center gap-2">
          <p class="truncate text-sm font-semibold">{pageTitle}</p>
          <Badge variant="outline">{pageLabel}</Badge>
          {#if statusMessage}
            <Badge variant={pageStatus === 'error' ? 'destructive' : 'outline'}>{statusMessage}</Badge>
          {/if}
        </div>
        <p class="text-muted-foreground mt-1 truncate text-xs">
          {selectedOrg ? `${selectedOrg.name}` : 'No organization selected'}
          {#if selectedProject}
            <span> / {selectedProject.name}</span>
          {/if}
          {#if !selectedProject}
            <span> / pick a project in the sidebar</span>
          {/if}
        </p>
      </div>
    </div>

    <div class="flex flex-wrap items-center gap-2">
      <div class="hidden items-center gap-2 lg:flex">
        <span class="border-border/70 bg-background/70 inline-flex items-center gap-2 rounded-full border px-3 py-1.5 text-xs">
          <RadioTower class="size-3.5" />
          {pageStatus}
        </span>
        <span class="border-border/70 bg-background/70 inline-flex items-center gap-2 rounded-full border px-3 py-1.5 text-xs">
          <FolderKanban class="size-3.5" />
          {runningAgentCount} running
        </span>
        <span class="border-border/70 bg-background/70 inline-flex items-center gap-2 rounded-full border px-3 py-1.5 text-xs">
          <Ticket class="size-3.5" />
          {ticketCount} tickets
        </span>
        <span class="border-border/70 bg-background/70 inline-flex items-center gap-2 rounded-full border px-3 py-1.5 text-xs">
          <Workflow class="size-3.5" />
          {workflowCount} workflows
        </span>
      </div>

      <Button type="button" variant="outline" onclick={onToggleDrawer}>
        <PanelRightOpen class="mr-2 size-4" />
        Context
      </Button>

      {#if selectedProject}
        <a
          href={`/board?org=${encodeURIComponent(selectedOrg?.id ?? '')}&project=${encodeURIComponent(selectedProject.id)}`}
          class="bg-foreground text-background inline-flex items-center rounded-full px-4 py-2 text-sm font-medium"
        >
          Open board
        </a>
      {:else}
        <span class="text-muted-foreground hidden text-xs lg:inline">Select a project to activate the workspace.</span>
      {/if}
    </div>
  </div>
</div>
