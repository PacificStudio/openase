<script lang="ts">
  import type { WorkflowRepositoryPrerequisite } from '../repository-prerequisite'
  import WorkflowRepositoryPrerequisiteCard from './workflow-repository-prerequisite-card.svelte'

  let {
    loading = false,
    prerequisite = null,
    settingsHref = null,
    loadError = '',
  }: {
    loading?: boolean
    prerequisite?: WorkflowRepositoryPrerequisite | null
    settingsHref?: string | null
    loadError?: string
  } = $props()
</script>

{#if loading}
  <div class="text-muted-foreground flex flex-1 items-center justify-center text-sm">
    Loading workflows…
  </div>
{:else if prerequisite && prerequisite.kind !== 'ready'}
  <div class="flex flex-1 p-4">
    <WorkflowRepositoryPrerequisiteCard {prerequisite} {settingsHref} />
  </div>
{:else if loadError}
  <div
    class="border-destructive/40 bg-destructive/10 text-destructive m-4 rounded-md border px-4 py-3 text-sm"
  >
    {loadError}
  </div>
{/if}
