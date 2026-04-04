<script lang="ts">
  import type { Project } from '$lib/api/contracts'
  import { projectPath } from '$lib/stores/app-context'

  let {
    orgId,
    projects,
  }: {
    orgId: string | null
    projects: Project[]
  } = $props()
</script>

{#if projects.length > 0}
  <div class="grid grid-cols-1 gap-4 lg:grid-cols-2">
    {#each projects as project (project.id)}
      <article class="border-border bg-card rounded-lg border p-5">
        <div class="flex items-start justify-between gap-3">
          <div class="space-y-1">
            <h3 class="text-foreground text-base font-semibold">{project.name}</h3>
            <p class="text-muted-foreground text-sm">
              {project.description || 'No project description yet.'}
            </p>
          </div>
          <span
            class="bg-muted text-muted-foreground rounded-full px-2.5 py-1 text-[11px] font-medium"
          >
            {project.status}
          </span>
        </div>

        <div class="mt-4 flex flex-wrap gap-2">
          <a
            href={orgId ? projectPath(orgId, project.id) : '/'}
            class="bg-primary text-primary-foreground inline-flex h-9 items-center rounded-md px-4 text-sm font-medium"
          >
            Open dashboard
          </a>
          <a
            href={orgId ? projectPath(orgId, project.id, 'tickets') : '/'}
            class="border-border hover:bg-accent inline-flex h-9 items-center rounded-md border px-4 text-sm font-medium transition-colors"
          >
            Open tickets
          </a>
        </div>
      </article>
    {/each}
  </div>
{:else}
  <div
    class="border-border bg-card text-muted-foreground rounded-lg border px-4 py-10 text-center text-sm"
  >
    No projects found in this organization yet. Create project above to open the first dashboard
    route.
  </div>
{/if}
