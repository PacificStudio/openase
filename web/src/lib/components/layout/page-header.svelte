<script lang="ts">
  import type { Snippet } from 'svelte'
  import type { ProjectSection } from '$lib/stores/app-context'
  import PageHelpButton from './page-help-button.svelte'

  let {
    title,
    description = '',
    actions,
    helpSection,
  }: {
    title: string
    description?: string
    actions?: Snippet
    helpSection?: ProjectSection
  } = $props()
</script>

<div
  class="border-border flex flex-col gap-3 border-b px-4 py-3 sm:flex-row sm:items-center sm:justify-between sm:px-6 sm:py-4"
>
  <div class="min-w-0">
    <div class="flex items-center gap-1.5">
      <h1 class="text-foreground truncate text-lg font-semibold">{title}</h1>
      {#if helpSection}
        <PageHelpButton section={helpSection} />
      {/if}
    </div>
    {#if description}
      <p class="text-muted-foreground mt-0.5 text-sm">{description}</p>
    {/if}
  </div>
  {#if actions}
    <div class="flex shrink-0 items-center gap-2">
      {@render actions()}
    </div>
  {/if}
</div>
