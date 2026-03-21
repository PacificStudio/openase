<script lang="ts">
  import {
    capabilityCatalog,
    capabilityStateClasses,
    capabilityStateLabel,
  } from '$lib/features/capabilities'

  const connectorsCapability = capabilityCatalog.connectorsSettings
</script>

<div class="max-w-lg space-y-4">
  <div class="flex items-center gap-2">
    <h2 class="text-foreground text-base font-semibold">Connectors</h2>
    <span
      class={`inline-flex items-center rounded-full border px-2 py-0.5 text-[11px] font-medium ${capabilityStateClasses(connectorsCapability.state)}`}
    >
      {capabilityStateLabel(connectorsCapability.state)}
    </span>
  </div>

  <p class="text-muted-foreground text-sm">{connectorsCapability.summary}</p>
</div>
