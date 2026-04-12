<script lang="ts">
  import { Label } from '$ui/label'
  import * as Select from '$ui/select'
  import type { ProviderReasoningCapability } from '../types'
  import {
    formatReasoningEffortLabel,
    providerDefaultReasoningValue,
    providerReasoningCapabilitySummary,
  } from '../provider-model-options'

  let {
    capability,
    selectedEffort = '',
    effectiveEffort = '',
    onValueChange,
  }: {
    capability: ProviderReasoningCapability | null | undefined
    selectedEffort?: string
    effectiveEffort?: string
    onValueChange?: (value: string) => void
  } = $props()
</script>

<div class="space-y-2">
  <Label>Reasoning preset</Label>
  {#if capability?.state === 'available'}
    <Select.Root
      type="single"
      value={selectedEffort.trim() || providerDefaultReasoningValue}
      onValueChange={(value) =>
        onValueChange?.(value === providerDefaultReasoningValue ? '' : (value ?? ''))}
    >
      <Select.Trigger class="w-full">
        {#if selectedEffort.trim()}
          {formatReasoningEffortLabel(selectedEffort.trim())}
        {:else if capability.defaultEffort}
          Model default ({formatReasoningEffortLabel(capability.defaultEffort)})
        {:else}
          Use model default
        {/if}
      </Select.Trigger>
      <Select.Content>
        <Select.Item value={providerDefaultReasoningValue}>
          Use model default
          {#if capability.defaultEffort}
            ({formatReasoningEffortLabel(capability.defaultEffort)})
          {/if}
        </Select.Item>
        {#each capability.supportedEfforts ?? [] as effort (effort)}
          <Select.Item value={effort}>{formatReasoningEffortLabel(effort)}</Select.Item>
        {/each}
      </Select.Content>
    </Select.Root>
    <p class="text-muted-foreground text-xs">
      {providerReasoningCapabilitySummary(capability)}
    </p>
    {#if effectiveEffort}
      <p class="text-muted-foreground text-xs">
        Effective effort: {formatReasoningEffortLabel(effectiveEffort)}.
      </p>
    {/if}
  {:else}
    <div
      class="border-border bg-muted/30 text-muted-foreground rounded-md border px-3 py-2 text-sm"
    >
      {providerReasoningCapabilitySummary(capability)}
    </div>
  {/if}
</div>
