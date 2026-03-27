<script lang="ts">
  import { GitBranch } from '@lucide/svelte'
  import { Button } from '$ui/button'
  import * as Card from '$ui/card'

  let {
    repoCount,
    settingsHref = null,
    onOpenRepositories,
  }: {
    repoCount: number
    settingsHref?: string | null
    onOpenRepositories?: (() => void) | undefined
  } = $props()

  const hasConfiguredRepos = $derived(repoCount > 0)
</script>

<Card.Root class="border-border/80 max-w-2xl">
  <Card.Header class="gap-3">
    <div class="bg-muted text-foreground flex size-10 items-center justify-center rounded-xl">
      <GitBranch class="size-5" />
    </div>
    <div class="space-y-1">
      <Card.Title>
        {hasConfiguredRepos ? 'Mark a primary repository first' : 'Primary repository required'}
      </Card.Title>
      <Card.Description>
        Workflow creation and harness editing need a project primary repository so OpenASE can
        resolve the default workspace and repo-scoped automation paths.
      </Card.Description>
    </div>
  </Card.Header>

  <Card.Content class="space-y-4">
    <p class="text-muted-foreground text-sm">
      {#if hasConfiguredRepos}
        This project already has repositories, but none is marked as primary.
      {:else}
        This project does not have any repositories yet.
      {/if}
      Add at least one primary repository before creating or editing workflows.
    </p>

    {#if onOpenRepositories}
      <Button type="button" onclick={onOpenRepositories}>Open Repositories</Button>
    {:else if settingsHref}
      <Button href={settingsHref}>Open Settings</Button>
    {/if}
  </Card.Content>
</Card.Root>
