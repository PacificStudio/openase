<script lang="ts">
  import { Building2, FolderKanban } from '@lucide/svelte'
  import ScrollPane from '$lib/components/layout/ScrollPane.svelte'
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

<div class="grid min-h-0 gap-4">
  <div class="min-h-0">
    <div class="mb-2 flex items-center gap-2 px-1">
      <Building2 class="text-muted-foreground size-4" />
      <p class="text-sm font-medium">Organizations</p>
      <span class="text-muted-foreground text-xs">{organizations.length}</span>
    </div>

    <ScrollPane class="max-h-44 space-y-2">
      {#if organizations.length === 0}
        <div
          class="text-muted-foreground border-border/70 bg-muted/35 rounded-2xl border border-dashed px-3 py-4 text-sm"
        >
          No organizations yet.
        </div>
      {:else}
        {#each organizations as organization}
          <button
            type="button"
            class={`w-full rounded-2xl border px-3 py-3 text-left text-sm transition ${
              organization.id === selectedOrgId
                ? 'border-foreground/25 bg-foreground text-background shadow-sm'
                : 'border-border/70 bg-background/60 hover:border-foreground/15 hover:bg-background'
            }`}
            onclick={() => onSelectOrganization?.(organization)}
          >
            <div class="flex items-center justify-between gap-2">
              <span class="font-medium">{organization.name}</span>
              {#if organization.id === selectedOrgId}
                <span class="rounded-full border border-background/20 px-2 py-0.5 text-[11px] text-background/75">
                  active
                </span>
              {/if}
            </div>
            <p
              class={`mt-1 truncate font-mono text-[11px] ${organization.id === selectedOrgId ? 'text-background/75' : 'text-muted-foreground'}`}
            >
              /{organization.slug}
            </p>
          </button>
        {/each}
      {/if}
    </ScrollPane>
  </div>

  <div class="min-h-0">
    <div class="mb-2 flex items-center gap-2 px-1">
      <FolderKanban class="text-muted-foreground size-4" />
      <p class="text-sm font-medium">Projects</p>
      <span class="text-muted-foreground text-xs">{projects.length}</span>
    </div>

    <ScrollPane class="max-h-64 space-y-2">
      {#if projects.length === 0}
        <div
          class="text-muted-foreground border-border/70 bg-muted/35 rounded-2xl border border-dashed px-3 py-4 text-sm"
        >
          Select an organization, then choose a project.
        </div>
      {:else}
        {#each projects as project}
          <button
            type="button"
            class={`w-full rounded-2xl border px-3 py-3 text-left text-sm transition ${
              project.id === selectedProjectId
                ? 'border-emerald-700/35 bg-emerald-950 text-white shadow-sm shadow-emerald-950/20'
                : 'border-border/70 bg-background/60 hover:border-foreground/15 hover:bg-background'
            }`}
            onclick={() => onSelectProject?.(project)}
          >
            <div class="flex items-center justify-between gap-2">
              <span class="font-medium">{project.name}</span>
              <span
                class={`rounded-full border px-2 py-0.5 text-[11px] ${
                  project.id === selectedProjectId
                    ? 'border-white/15 text-white/75'
                    : 'border-border/70 text-muted-foreground'
                }`}
              >
                {project.status}
              </span>
            </div>
            <p
              class={`mt-1 truncate font-mono text-[11px] ${project.id === selectedProjectId ? 'text-white/75' : 'text-muted-foreground'}`}
            >
              /{project.slug}
            </p>
          </button>
        {/each}
      {/if}
    </ScrollPane>
  </div>
</div>
