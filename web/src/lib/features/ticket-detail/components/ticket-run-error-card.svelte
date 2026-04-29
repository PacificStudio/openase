<script lang="ts">
  import type { TicketRun } from '../types'

  let { run }: { run: TicketRun | null } = $props()
</script>

{#if run?.lastError}
  <section class="space-y-1.5">
    <div class="text-destructive text-[10px] font-medium tracking-wider uppercase">
      Error details
    </div>
    <div class="border-destructive/30 bg-destructive/5 rounded-md border px-3 py-2.5">
      <p class="text-destructive text-xs leading-relaxed break-words">{run.lastError}</p>
    </div>
  </section>
{/if}
