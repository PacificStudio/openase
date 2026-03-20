<script lang="ts">
  import { Building2, FolderKanban } from '@lucide/svelte'
  import { Badge } from '$lib/components/ui/badge'
  import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
  } from '$lib/components/ui/card'
  import type { Organization, Project } from '$lib/types/workspace-shell'

  let {
    organizations = [],
    projects = [],
    selectedOrgId = '',
    selectedProjectId = '',
    onSelectOrganization,
    onSelectProject,
  }: {
    organizations?: Organization[]
    projects?: Project[]
    selectedOrgId?: string
    selectedProjectId?: string
    onSelectOrganization?: (organization: Organization) => void
    onSelectProject?: (project: Project) => void
  } = $props()
</script>

<div class="space-y-6">
  <Card class="border-border/80 bg-background/80 backdrop-blur">
    <CardHeader>
      <div class="flex items-center justify-between gap-3">
        <div>
          <CardTitle class="flex items-center gap-2">
            <Building2 class="size-4" />
            <span>Organizations</span>
          </CardTitle>
          <CardDescription>Choose the workspace boundary.</CardDescription>
        </div>
        <Badge variant="outline">{organizations.length}</Badge>
      </div>
    </CardHeader>
    <CardContent class="space-y-3">
      {#if organizations.length === 0}
        <div
          class="text-muted-foreground border-border/70 bg-muted/35 rounded-3xl border border-dashed px-4 py-5 text-sm"
        >
          No organizations yet.
        </div>
      {:else}
        {#each organizations as organization}
          <button
            type="button"
            class={`w-full rounded-3xl border px-4 py-4 text-left transition ${
              organization.id === selectedOrgId
                ? 'border-foreground/30 bg-foreground text-background shadow-lg shadow-black/10'
                : 'border-border/70 bg-background/60 hover:border-foreground/15 hover:bg-background'
            }`}
            onclick={() => onSelectOrganization?.(organization)}
          >
            <div class="flex items-start justify-between gap-3">
              <div>
                <p class="text-sm font-semibold">{organization.name}</p>
                <p
                  class={`mt-1 font-mono text-xs ${organization.id === selectedOrgId ? 'text-background/75' : 'text-muted-foreground'}`}
                >
                  /{organization.slug}
                </p>
              </div>
              {#if organization.id === selectedOrgId}
                <Badge variant="secondary">selected</Badge>
              {/if}
            </div>
          </button>
        {/each}
      {/if}
    </CardContent>
  </Card>

  <Card class="border-border/80 bg-background/80 backdrop-blur">
    <CardHeader>
      <div class="flex items-center justify-between gap-3">
        <div>
          <CardTitle class="flex items-center gap-2">
            <FolderKanban class="size-4" />
            <span>Projects</span>
          </CardTitle>
          <CardDescription>Switch the active board and dashboard.</CardDescription>
        </div>
        <Badge variant="outline">{projects.length}</Badge>
      </div>
    </CardHeader>
    <CardContent class="space-y-3">
      {#if projects.length === 0}
        <div
          class="text-muted-foreground border-border/70 bg-muted/35 rounded-3xl border border-dashed px-4 py-5 text-sm"
        >
          Select an organization, then create a project.
        </div>
      {:else}
        {#each projects as project}
          <button
            type="button"
            class={`w-full rounded-3xl border px-4 py-4 text-left transition ${
              project.id === selectedProjectId
                ? 'border-emerald-700/35 bg-emerald-950 text-white shadow-lg shadow-emerald-950/20'
                : 'border-border/70 bg-background/60 hover:border-foreground/15 hover:bg-background'
            }`}
            onclick={() => onSelectProject?.(project)}
          >
            <div class="flex items-start justify-between gap-3">
              <div>
                <p class="text-sm font-semibold">{project.name}</p>
                <p
                  class={`mt-1 font-mono text-xs ${project.id === selectedProjectId ? 'text-white/70' : 'text-muted-foreground'}`}
                >
                  /{project.slug}
                </p>
              </div>
              <Badge variant={project.id === selectedProjectId ? 'secondary' : 'outline'}>
                {project.status}
              </Badge>
            </div>
            <p
              class={`mt-3 text-sm leading-6 ${project.id === selectedProjectId ? 'text-white/80' : 'text-muted-foreground'}`}
            >
              {project.description || 'No description yet.'}
            </p>
          </button>
        {/each}
      {/if}
    </CardContent>
  </Card>
</div>
