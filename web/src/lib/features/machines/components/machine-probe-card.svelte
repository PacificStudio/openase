<script lang="ts">
  import { Badge } from '$ui/badge'
  import { formatRelativeTime } from '$lib/utils'
  import {
    machineDetectedArchLabel,
    machineDetectedOSLabel,
    machineDetectionBadgeClass,
    machineDetectionStatusLabel,
  } from '../model'
  import { friendlyTransportLabel } from '../machine-setup'
  import type { MachineProbeResult } from '../types'

  let { probe }: { probe: MachineProbeResult } = $props()
</script>

<div class="border-border bg-card rounded-xl border px-4 py-4">
  <div class="flex items-start justify-between gap-3">
    <div class="space-y-3">
      <h4 class="text-foreground text-sm font-semibold">Latest connection test</h4>
      <p class="text-muted-foreground mt-1 text-xs">
        {formatRelativeTime(probe.checked_at)}
      </p>
      <div class="flex flex-wrap items-center gap-2">
        <Badge variant="outline">{friendlyTransportLabel(probe.transport)}</Badge>
        <Badge variant="secondary">{machineDetectedOSLabel(probe.detected_os)}</Badge>
        <Badge variant="secondary">{machineDetectedArchLabel(probe.detected_arch)}</Badge>
        <Badge variant="outline" class={machineDetectionBadgeClass(probe.detection_status)}>
          {machineDetectionStatusLabel(probe.detection_status)}
        </Badge>
      </div>
      <p class="text-muted-foreground text-xs">
        {probe.detection_message ||
          'Connection tests can still pass even when platform detection is partial.'}
      </p>
    </div>
  </div>

  <pre
    class="bg-muted/50 text-foreground mt-4 overflow-x-auto rounded-lg px-3 py-3 text-xs whitespace-pre-wrap">{probe.output ||
      'Probe completed without output.'}</pre>
</div>
