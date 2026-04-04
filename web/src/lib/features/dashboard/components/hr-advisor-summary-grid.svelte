<script lang="ts">
  import type { HRAdvisorSummary } from '$lib/api/contracts'

  let { summary }: { summary: HRAdvisorSummary } = $props()
</script>

<div class="grid grid-cols-2 gap-3 text-sm">
  <div class="rounded-md border border-emerald-500/20 bg-emerald-500/5 px-3 py-2">
    <div class="text-muted-foreground text-[11px] tracking-wide uppercase">Open tickets</div>
    <div class="text-foreground mt-1 text-lg font-semibold">{summary.open_tickets}</div>
  </div>
  <div class="rounded-md border border-amber-500/20 bg-amber-500/5 px-3 py-2">
    <div class="text-muted-foreground text-[11px] tracking-wide uppercase">Blocked</div>
    <div class="text-foreground mt-1 text-lg font-semibold">{summary.blocked_tickets}</div>
  </div>
  <div class="rounded-md border border-sky-500/20 bg-sky-500/5 px-3 py-2">
    <div class="text-muted-foreground text-[11px] tracking-wide uppercase">Coding</div>
    <div class="text-foreground mt-1 text-lg font-semibold">{summary.coding_tickets}</div>
  </div>
  <div class="rounded-md border border-violet-500/20 bg-violet-500/5 px-3 py-2">
    <div class="text-muted-foreground text-[11px] tracking-wide uppercase">Active agents</div>
    <div class="text-foreground mt-1 text-lg font-semibold">{summary.active_agents}</div>
  </div>
</div>
