<script lang="ts">
  import WorkflowRepositoryPrerequisiteCard from './workflow-repository-prerequisite-card.svelte'

  let {
    loading = false,
    missingPrimaryRepo = false,
    repoCount = 0,
    settingsHref = null,
    loadError = '',
  }: {
    loading?: boolean
    missingPrimaryRepo?: boolean
    repoCount?: number
    settingsHref?: string | null
    loadError?: string
  } = $props()
</script>

{#if loading}
  <div class="text-muted-foreground flex flex-1 items-center justify-center text-sm">
    Loading workflows…
  </div>
{:else if missingPrimaryRepo}
  <div class="flex flex-1 p-4">
    <WorkflowRepositoryPrerequisiteCard {repoCount} {settingsHref} />
  </div>
{:else if loadError}
  <div
    class="border-destructive/40 bg-destructive/10 text-destructive m-4 rounded-md border px-4 py-3 text-sm"
  >
    {loadError}
  </div>
{/if}
