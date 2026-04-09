<script lang="ts">
  import type { SecurityDocumentationLink } from '$lib/api/contracts'
  import { ExternalLink, LifeBuoy } from '@lucide/svelte'

  let { docs = [] }: { docs?: SecurityDocumentationLink[] } = $props()
</script>

<div class="space-y-2">
  <div class="flex items-center gap-2 px-0.5">
    <LifeBuoy class="text-muted-foreground size-4" />
    <h4 class="text-sm font-semibold">Mode switch help</h4>
  </div>
  <div class="grid gap-3 lg:grid-cols-3">
    {#each docs as doc (doc.href)}
      <a
        class="border-border bg-card hover:border-border/80 hover:bg-muted/40 flex flex-col gap-2 rounded-lg border p-4 transition-colors"
        href={doc.href}
        target="_blank"
        rel="noreferrer"
      >
        <div class="flex items-start justify-between gap-3">
          <div class="text-sm font-medium">{doc.title}</div>
          <ExternalLink class="text-muted-foreground mt-0.5 size-3.5 shrink-0" />
        </div>
        <p class="text-muted-foreground text-xs leading-relaxed">{doc.summary}</p>
      </a>
    {/each}
  </div>
</div>
