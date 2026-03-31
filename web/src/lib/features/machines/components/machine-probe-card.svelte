<script lang="ts">
  import { Badge } from '$ui/badge'
  import { formatRelativeTime } from '$lib/utils'
  import type { MachineProbeResult } from '../types'

  let { probe }: { probe: MachineProbeResult } = $props()
</script>

<div class="border-border bg-card rounded-xl border px-4 py-4">
  <div class="flex items-center justify-between gap-3">
    <div>
      <h4 class="text-foreground text-sm font-semibold">Latest connection test</h4>
      <p class="text-muted-foreground mt-1 text-xs">
        {formatRelativeTime(probe.checked_at)}
      </p>
    </div>
    <Badge variant="outline">{probe.transport}</Badge>
  </div>

  <pre
    class="bg-muted/50 text-foreground mt-4 overflow-x-auto rounded-lg px-3 py-3 text-xs whitespace-pre-wrap">{probe.output ||
      'Probe completed without output.'}</pre>
</div>
