<script lang="ts">
  let {
    orgName,
    projectCount,
    providerCount,
  }: {
    orgName: string
    projectCount: number
    providerCount: number
  } = $props()
</script>

<div class="grid grid-cols-1 gap-4 md:grid-cols-3">
  <div class="border-border bg-card rounded-lg border p-4">
    <p class="text-muted-foreground text-xs tracking-[0.2em] uppercase">Organization</p>
    <p class="text-foreground mt-2 text-lg font-semibold">{orgName}</p>
    <p class="text-muted-foreground mt-1 text-sm">Stable context now lives in the URL.</p>
  </div>
  <div class="border-border bg-card rounded-lg border p-4">
    <p class="text-muted-foreground text-xs tracking-[0.2em] uppercase">Projects</p>
    <p class="text-foreground mt-2 text-lg font-semibold">{projectCount}</p>
    <p class="text-muted-foreground mt-1 text-sm">
      Pick a project to open its dashboard, board, agents, and settings under one route model.
    </p>
  </div>
  <div class="border-border bg-card rounded-lg border p-4">
    <p class="text-muted-foreground text-xs tracking-[0.2em] uppercase">Providers</p>
    <p class="text-foreground mt-2 text-lg font-semibold">{providerCount}</p>
    <p class="text-muted-foreground mt-1 text-sm">
      Provider configuration remains attached to the selected organization context.
    </p>
  </div>
</div>
