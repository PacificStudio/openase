<script lang="ts">
  import {
    capabilityStateClasses,
    capabilityStateLabel,
    getSettingsSectionCapability,
  } from '$lib/features/capabilities'
  import type { SettingsSection } from '../types'

  let { section, title }: { section: SettingsSection; title: string } = $props()

  const capability = $derived(getSettingsSectionCapability(section))
</script>

<div class="max-w-lg space-y-4">
  <div class="flex items-center gap-2">
    <h2 class="text-foreground text-base font-semibold">{title}</h2>
    <span
      class={`inline-flex items-center rounded-full border px-2 py-0.5 text-[11px] font-medium ${capabilityStateClasses(capability.state)}`}
    >
      {capabilityStateLabel(capability.state)}
    </span>
  </div>

  <p class="text-muted-foreground text-sm">{capability.summary}</p>
</div>
