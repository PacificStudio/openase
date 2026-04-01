<script lang="ts">
  let {
    status = '',
    rows = [],
    notes = [],
  }: {
    status?: string
    rows?: [string, string][]
    notes?: string[]
  } = $props()
</script>

{#if status}
  <div class="space-y-2 md:col-span-2">
    <p class="text-muted-foreground text-xs">{status}</p>
    {#if rows.length > 0}
      <div
        class="border-border bg-muted/30 grid gap-2 rounded-md border p-3 text-xs md:grid-cols-3"
      >
        {#each rows as row (`${row[0]}:${row[1]}`)}
          <div>
            <div class="text-muted-foreground">{row[0]}</div>
            <div class="font-medium">{row[1]} USD / 1M</div>
          </div>
        {/each}
      </div>
    {/if}
    {#if notes.length > 0}
      <p class="text-muted-foreground text-xs">{notes.join(' ')}</p>
    {/if}
  </div>
{/if}
