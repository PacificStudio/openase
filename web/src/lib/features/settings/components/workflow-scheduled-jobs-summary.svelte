<script lang="ts">
  let {
    total,
    enabled,
  }: {
    total: number
    enabled: number
  } = $props()
</script>

<div
  class="border-border/60 bg-card/60 mb-3 flex flex-wrap items-center gap-x-5 gap-y-1 rounded-xl border px-4 py-3 text-sm"
>
  <span class="text-foreground font-medium">{total} job{total !== 1 ? 's' : ''}</span>
  <span class="text-muted-foreground">{enabled} enabled</span>
</div>
