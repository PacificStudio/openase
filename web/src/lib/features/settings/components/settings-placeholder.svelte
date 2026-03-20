<script lang="ts">
  import {
    capabilityCatalog,
    capabilityStateClasses,
    capabilityStateLabel,
    settingsCapabilityBySection,
  } from '$lib/features/capabilities'
  import type { SettingsSection } from '../types'

  let { section, title }: { section: SettingsSection; title: string } = $props()

  const capabilityKey = $derived(settingsCapabilityBySection[section])
  const capability = $derived(capabilityKey ? capabilityCatalog[capabilityKey] : null)
</script>

<div class="max-w-lg space-y-4">
  <div class="flex items-center gap-2">
    <h2 class="text-foreground text-base font-semibold">{title}</h2>
    {#if capability}
      <span
        class={`inline-flex items-center rounded-full border px-2 py-0.5 text-[11px] font-medium ${capabilityStateClasses(capability.state)}`}
      >
        {capabilityStateLabel(capability.state)}
      </span>
    {/if}
  </div>

  <p class="text-muted-foreground text-sm">
    {capability?.summary ?? 'This settings section has not been classified yet.'}
  </p>
</div>
