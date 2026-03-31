<script lang="ts">
  import { GitBranch } from '@lucide/svelte'
  import type { WorkflowRepositoryPrerequisite } from '../repository-prerequisite'
  import { Button } from '$ui/button'
  import * as Card from '$ui/card'

  let {
    prerequisite,
    settingsHref = null,
    onOpenRepositories,
  }: {
    prerequisite: Exclude<WorkflowRepositoryPrerequisite, { kind: 'ready' }>
    settingsHref?: string | null
    onOpenRepositories?: (() => void) | undefined
  } = $props()

  const hasConfiguredRepos = $derived(prerequisite.repoCount > 0)

  const title = $derived(
    hasConfiguredRepos ? 'Mark a primary repository first' : 'Primary repository required',
  )

  const summary = $derived(
    hasConfiguredRepos
      ? 'This project already has repositories, but none is marked as primary.'
      : 'This project does not have any repositories yet.',
  )
</script>

<Card.Root class="border-border/80 max-w-2xl">
  <Card.Header class="gap-3">
    <div class="bg-muted text-foreground flex size-10 items-center justify-center rounded-xl">
      <GitBranch class="size-5" />
    </div>
    <div class="space-y-1">
      <Card.Title>{title}</Card.Title>
      <Card.Description>
        Workflow creation and harness editing require a primary repository binding.
      </Card.Description>
    </div>
  </Card.Header>

  <Card.Content class="space-y-4">
    <p class="text-muted-foreground text-sm">{summary}</p>

    <p class="text-muted-foreground text-sm">
      Add at least one primary repository before creating or editing workflows.
    </p>

    {#if onOpenRepositories}
      <Button type="button" onclick={onOpenRepositories}>Open Repositories</Button>
    {:else if settingsHref}
      <Button href={settingsHref}>Open Settings</Button>
    {/if}
  </Card.Content>
</Card.Root>
