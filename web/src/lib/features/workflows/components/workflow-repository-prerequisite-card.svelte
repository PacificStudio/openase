<script lang="ts">
  import { GitBranch } from '@lucide/svelte'
  import type { WorkflowRepositoryPrerequisite } from '../data'
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

  const title = $derived.by(() => {
    if (prerequisite.kind === 'missing_primary_repo') {
      return hasConfiguredRepos ? 'Mark a primary repository first' : 'Primary repository required'
    }

    switch (prerequisite.action) {
      case 'prepare_primary_mirror':
        return 'Prepare the primary mirror'
      case 'wait_for_primary_mirror':
        return 'Primary mirror is still preparing'
      case 'sync_primary_mirror':
        return 'Primary mirror needs attention'
      default:
        return 'Primary mirror is not ready'
    }
  })

  const summary = $derived.by(() => {
    if (prerequisite.kind === 'missing_primary_repo') {
      return hasConfiguredRepos
        ? 'This project already has repositories, but none is marked as primary.'
        : 'This project does not have any repositories yet.'
    }

    const repoLabel = prerequisite.primaryRepoName || 'The primary repository'
    switch (prerequisite.action) {
      case 'prepare_primary_mirror':
        return `${repoLabel} is bound, but no ready mirror is available yet. Prepare a mirror before creating or editing workflows.`
      case 'wait_for_primary_mirror':
        return `${repoLabel} is bound, but its mirror is currently ${prerequisite.mirrorState}. Wait for mirror preparation to finish.`
      case 'sync_primary_mirror':
        return `${repoLabel} is bound, but its mirror is ${prerequisite.mirrorState}. Repair or resync the mirror before continuing.`
      default:
        return `${repoLabel} is bound, but its primary mirror is not ready yet.`
    }
  })
</script>

<Card.Root class="border-border/80 max-w-2xl">
  <Card.Header class="gap-3">
    <div class="bg-muted text-foreground flex size-10 items-center justify-center rounded-xl">
      <GitBranch class="size-5" />
    </div>
    <div class="space-y-1">
      <Card.Title>{title}</Card.Title>
      <Card.Description>
        Workflow creation and harness editing require a ready primary mirror, not just a bound
        primary repository.
      </Card.Description>
    </div>
  </Card.Header>

  <Card.Content class="space-y-4">
    <p class="text-muted-foreground text-sm">{summary}</p>

    {#if prerequisite.kind === 'missing_primary_repo'}
      <p class="text-muted-foreground text-sm">
        Add at least one primary repository before creating or editing workflows.
      </p>
    {/if}

    {#if prerequisite.kind === 'primary_mirror_not_ready'}
      <dl class="text-muted-foreground grid gap-2 text-sm">
        <div class="flex items-start justify-between gap-4">
          <dt>Mirror state</dt>
          <dd class="text-foreground font-medium">{prerequisite.mirrorState}</dd>
        </div>
        <div class="flex items-start justify-between gap-4">
          <dt>Known mirrors</dt>
          <dd class="text-foreground font-medium">{prerequisite.mirrorCount}</dd>
        </div>
      </dl>

      {#if prerequisite.mirrorLastError}
        <div class="border-destructive/30 bg-destructive/5 rounded-lg border px-3 py-2 text-sm">
          <p class="text-foreground font-medium">Last mirror error</p>
          <p class="text-muted-foreground mt-1 break-words">{prerequisite.mirrorLastError}</p>
        </div>
      {/if}
    {/if}

    {#if onOpenRepositories}
      <Button type="button" onclick={onOpenRepositories}>Open Repositories</Button>
    {:else if settingsHref}
      <Button href={settingsHref}>Open Settings</Button>
    {/if}
  </Card.Content>
</Card.Root>
