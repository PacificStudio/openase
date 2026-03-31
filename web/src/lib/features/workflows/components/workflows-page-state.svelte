<script lang="ts">
  let {
    loading = false,
    loadError = '',
  }: {
    loading?: boolean
    loadError?: string
  } = $props()
</script>

{#if loading}
  <div class="text-muted-foreground flex flex-1 items-center justify-center text-sm">
    Loading workflows…
  </div>
{:else if loadError}
  <div
    class="border-destructive/40 bg-destructive/10 text-destructive m-4 rounded-md border px-4 py-3 text-sm"
  >
    {loadError}
  </div>
{/if}
