<script lang="ts">
  let { metadata }: { metadata: Record<string, unknown> } = $props()
</script>

{#if Object.keys(metadata).length > 0}
  <pre
    class="border-border/70 bg-muted/25 text-muted-foreground mt-3 overflow-x-auto rounded-2xl border p-3 text-[11px]">{JSON.stringify(
      metadata,
      null,
      2,
    )}</pre>
{/if}
