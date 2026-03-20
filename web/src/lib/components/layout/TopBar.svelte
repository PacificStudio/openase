<script lang="ts">
  import { Menu, PanelsTopLeft, Sparkles } from '@lucide/svelte'
  import { Badge } from '$lib/components/ui/badge'
  import { Button } from '$lib/components/ui/button'
  import type { Organization, Project } from '$lib/types/workspace-shell'

  let {
    selectedOrg = null,
    selectedProject = null,
    notice = '',
    errorMessage = '',
    onToggleDrawer,
  }: {
    selectedOrg?: Organization | null
    selectedProject?: Project | null
    notice?: string
    errorMessage?: string
    onToggleDrawer?: () => void
  } = $props()
</script>

<div class="space-y-4">
  <div
    class="border-border/70 bg-background/80 rounded-[2rem] border px-5 py-5 shadow-sm backdrop-blur"
  >
    <div class="flex flex-wrap items-start justify-between gap-4">
      <div class="space-y-3">
        <div class="flex flex-wrap items-center gap-2">
          <Badge variant="outline">Feature-first shell</Badge>
          <Badge variant="outline">Dashboard</Badge>
          <Badge variant="outline">Board</Badge>
          <Badge variant="outline">SSE streams</Badge>
        </div>

        <div class="space-y-2">
          <p class="text-muted-foreground text-xs font-medium tracking-[0.28em] uppercase">
            OpenASE control plane
          </p>
          <div class="flex flex-wrap items-center gap-3">
            <h1 class="text-3xl font-semibold tracking-[-0.05em] text-balance sm:text-4xl">
              Project shell, live dashboard, and board now ship as separate features.
            </h1>
          </div>
          <p class="text-muted-foreground max-w-3xl text-sm leading-7 sm:text-base">
            Keep routing tickets from the board, watch agent telemetry update through SSE, and edit
            workflow settings from the drawer without stuffing page logic back into the route file.
          </p>
        </div>
      </div>

      <div class="flex flex-wrap items-center gap-3">
        <div
          class="border-border/70 bg-background/70 hidden rounded-2xl border px-4 py-3 text-sm lg:block"
        >
          <div class="flex items-center gap-2 font-medium">
            <PanelsTopLeft class="size-4" />
            <span>{selectedProject?.name ?? 'No project selected'}</span>
          </div>
          <p class="text-muted-foreground mt-1 text-xs">
            {selectedOrg ? `${selectedOrg.name} / ${selectedOrg.slug}` : 'Select an organization'}
          </p>
        </div>

        <Button type="button" variant="outline" class="lg:hidden" onclick={onToggleDrawer}>
          <Menu class="mr-2 size-4" />
          Drawer
        </Button>
      </div>
    </div>
  </div>

  {#if notice}
    <div
      class="rounded-3xl border border-emerald-500/30 bg-emerald-500/10 px-5 py-4 text-sm text-emerald-950"
    >
      <div class="flex items-center gap-2">
        <Sparkles class="size-4" />
        <span>{notice}</span>
      </div>
    </div>
  {/if}

  {#if errorMessage}
    <div
      class="text-destructive border-destructive/25 bg-destructive/10 rounded-3xl border px-5 py-4 text-sm"
    >
      {errorMessage}
    </div>
  {/if}
</div>
