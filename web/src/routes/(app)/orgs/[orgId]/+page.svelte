<script lang="ts">
  import { organizationPath, projectPath } from '$lib/features/app-shell/context'
  import type { PageData } from './$types'

  let { data }: { data: PageData } = $props()

  const currentOrg = $derived(data.currentOrg)
  const projects = $derived(data.projects)
</script>

<svelte:head>
  <title>{currentOrg?.name ?? 'Organization'} - OpenASE</title>
</svelte:head>

<div class="mx-auto flex w-full max-w-6xl flex-col gap-6 px-6 py-6">
  <div class="flex flex-col gap-2">
    <p class="text-muted-foreground text-sm">
      <a href="/" class="hover:text-foreground transition-colors">Workspace</a>
      <span class="mx-2">/</span>
      <a href={currentOrg ? organizationPath(currentOrg.id) : '/'} class="hover:text-foreground transition-colors">
        {currentOrg?.name ?? 'Organization'}
      </a>
    </p>
    <div>
      <h1 class="text-foreground text-2xl font-semibold">Dashboard</h1>
      <p class="text-muted-foreground mt-1 text-sm">
        Organization overview and project entry points for the current workspace.
      </p>
    </div>
  </div>

  <div class="grid grid-cols-1 gap-4 md:grid-cols-3">
    <div class="border-border bg-card rounded-lg border p-4">
      <p class="text-muted-foreground text-xs uppercase tracking-[0.2em]">Organization</p>
      <p class="text-foreground mt-2 text-lg font-semibold">{currentOrg?.name ?? 'Unknown org'}</p>
      <p class="text-muted-foreground mt-1 text-sm">Stable context now lives in the URL.</p>
    </div>
    <div class="border-border bg-card rounded-lg border p-4">
      <p class="text-muted-foreground text-xs uppercase tracking-[0.2em]">Projects</p>
      <p class="text-foreground mt-2 text-lg font-semibold">{projects.length}</p>
      <p class="text-muted-foreground mt-1 text-sm">
        Pick a project to open its dashboard, board, agents, and settings under one route model.
      </p>
    </div>
    <div class="border-border bg-card rounded-lg border p-4">
      <p class="text-muted-foreground text-xs uppercase tracking-[0.2em]">Providers</p>
      <p class="text-foreground mt-2 text-lg font-semibold">{data.providers.length}</p>
      <p class="text-muted-foreground mt-1 text-sm">
        Provider configuration remains attached to the selected organization context.
      </p>
    </div>
  </div>

  <section class="space-y-4">
    <div>
      <h2 class="text-foreground text-lg font-semibold">Projects</h2>
      <p class="text-muted-foreground mt-1 text-sm">
        Use direct links or the top-bar switcher to move between projects.
      </p>
    </div>

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
              <span class="bg-muted text-muted-foreground rounded-full px-2.5 py-1 text-[11px] font-medium">
                {project.status}
              </span>
            </div>

            <div class="mt-4 flex flex-wrap gap-2">
              <a
                href={currentOrg ? projectPath(currentOrg.id, project.id) : '/'}
                class="bg-primary text-primary-foreground inline-flex h-9 items-center rounded-md px-4 text-sm font-medium"
              >
                Open dashboard
              </a>
              <a
                href={currentOrg ? projectPath(currentOrg.id, project.id, 'board') : '/'}
                class="border-border hover:bg-accent inline-flex h-9 items-center rounded-md border px-4 text-sm font-medium transition-colors"
              >
                Open board
              </a>
            </div>
          </article>
        {/each}
      </div>
    {:else}
      <div class="border-border bg-card text-muted-foreground rounded-lg border px-4 py-10 text-center text-sm">
        No projects found in this organization yet.
      </div>
    {/if}
  </section>
</div>
