<script lang="ts">
  import SurfacePanel from '$lib/components/layout/SurfacePanel.svelte'
  import { Badge } from '$lib/components/ui/badge'
  import { getWorkspaceContext } from '$lib/features/workspace'

  const workspace = getWorkspaceContext()
</script>

<svelte:head>
  <title>Settings · OpenASE</title>
</svelte:head>

<div class="space-y-4">
  <SurfacePanel>
    {#snippet header()}
      <div>
        <p class="text-base font-semibold">Project configuration</p>
        <p class="text-muted-foreground mt-1 text-sm leading-6">
          Keep runtime views separate from project configuration. This settings surface collects
          repo wiring, workflow defaults, notification delivery, and future safety controls.
        </p>
      </div>
    {/snippet}

    <div class="grid gap-4 p-4 lg:grid-cols-3">
      <div class="border-border/70 bg-background/60 rounded-3xl border p-4">
        <p class="text-muted-foreground text-xs tracking-[0.22em] uppercase">Project</p>
        <p class="mt-3 text-lg font-semibold">
          {workspace.state.selectedProject?.name ?? 'No project selected'}
        </p>
        <div class="mt-3 flex flex-wrap gap-2">
          <Badge variant="outline">{workspace.state.workflows.length} workflows</Badge>
          <Badge variant="outline">{workspace.board.statuses.length} statuses</Badge>
        </div>
      </div>

      <div class="border-border/70 bg-background/60 rounded-3xl border p-4">
        <p class="text-muted-foreground text-xs tracking-[0.22em] uppercase">Organization</p>
        <p class="mt-3 text-lg font-semibold">
          {workspace.state.selectedOrg?.name ?? 'No organization selected'}
        </p>
        <p class="text-muted-foreground mt-2 text-sm">
          Context switching stays in the left rail so settings never push navigation off-screen.
        </p>
      </div>

      <div class="border-border/70 bg-background/60 rounded-3xl border p-4">
        <p class="text-muted-foreground text-xs tracking-[0.22em] uppercase">Status</p>
        <p class="mt-3 text-lg font-semibold">
          {workspace.state.errorMessage ? 'Needs attention' : 'Ready'}
        </p>
        <p class="text-muted-foreground mt-2 text-sm">
          Use the connectors and notifications sections for the current vertical slice.
        </p>
      </div>
    </div>
  </SurfacePanel>

  <SurfacePanel>
    {#snippet header()}
      <div>
        <p class="text-base font-semibold">Settings roadmap</p>
        <p class="text-muted-foreground mt-1 text-sm">
          Remaining sections are intentionally placed now so configuration no longer lives as
          unrelated top-level pages.
        </p>
      </div>
    {/snippet}

    <div class="grid gap-3 p-4 md:grid-cols-2">
      <div class="border-border/70 bg-background/60 rounded-3xl border p-4">
        <p class="font-medium">Repositories</p>
        <p class="text-muted-foreground mt-2 text-sm">Repo scopes, default branches, and PR policy.</p>
      </div>
      <div class="border-border/70 bg-background/60 rounded-3xl border p-4">
        <p class="font-medium">Agents</p>
        <p class="text-muted-foreground mt-2 text-sm">Provider defaults, safety policy, and budgets.</p>
      </div>
      <div class="border-border/70 bg-background/60 rounded-3xl border p-4">
        <p class="font-medium">Status Columns</p>
        <p class="text-muted-foreground mt-2 text-sm">Board topology, WIP budget, and lane semantics.</p>
      </div>
      <div class="border-border/70 bg-background/60 rounded-3xl border p-4">
        <p class="font-medium">Security</p>
        <p class="text-muted-foreground mt-2 text-sm">Approval boundaries and protected operations.</p>
      </div>
    </div>
  </SurfacePanel>
</div>
