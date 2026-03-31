<script lang="ts">
  import { CircleCheck, Info } from '@lucide/svelte'
  import type { RepositoryBindingsReadiness } from '../repositories-readiness'

  let {
    readiness,
  }: {
    readiness: RepositoryBindingsReadiness
  } = $props()
</script>

{#if readiness.kind === 'missing_repo'}
  <div
    class="flex items-center gap-3 rounded-lg border border-amber-500/30 bg-amber-500/5 px-4 py-3"
  >
    <Info class="size-4 shrink-0 text-amber-600" />
    <p class="text-foreground flex-1 text-sm">
      No repositories configured yet. Add a repository binding before relying on repo-scoped
      automation.
    </p>
  </div>
{:else}
  <div class="border-border bg-muted/30 flex items-center gap-3 rounded-lg border px-4 py-3">
    <CircleCheck class="size-4 shrink-0 text-emerald-500" />
    <span class="text-muted-foreground text-sm">
      {readiness.repoCount} repos configured for this project.
    </span>
  </div>
{/if}
