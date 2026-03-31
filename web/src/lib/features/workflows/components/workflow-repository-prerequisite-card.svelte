<script lang="ts">
  import { GitBranch } from '@lucide/svelte'
  import type { WorkflowRepositoryPrerequisite } from '../repository-prerequisite'
  import * as Card from '$ui/card'

  let {
    prerequisite,
  }: {
    prerequisite: WorkflowRepositoryPrerequisite
  } = $props()
</script>

<Card.Root class="border-border/80 max-w-2xl">
  <Card.Header class="gap-3">
    <div class="bg-muted text-foreground flex size-10 items-center justify-center rounded-xl">
      <GitBranch class="size-5" />
    </div>
    <div class="space-y-1">
      <Card.Title>Repository Context</Card.Title>
      <Card.Description>
        Repo-specific execution resolves from explicit repo scopes or the only bound repository in
        single-repo projects.
      </Card.Description>
    </div>
  </Card.Header>

  <Card.Content class="space-y-4">
    <p class="text-muted-foreground text-sm">
      This project currently has {prerequisite.repoCount} bound
      {prerequisite.repoCount === 1 ? ' repository' : ' repositories'}.
    </p>
  </Card.Content>
</Card.Root>
