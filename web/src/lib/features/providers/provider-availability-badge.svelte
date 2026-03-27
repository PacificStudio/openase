<script lang="ts">
  import { Badge } from '$ui/badge'
  import * as Tooltip from '$ui/tooltip'
  import {
    providerAvailabilityBadgeVariant,
    providerAvailabilityCheckedAtText,
    providerAvailabilityDescription,
    providerAvailabilityHeadline,
    providerAvailabilityLabel,
  } from './availability'

  let {
    availabilityState,
    availabilityReason = null,
    availabilityCheckedAt = null,
    class: className,
  }: {
    availabilityState: string
    availabilityReason?: string | null
    availabilityCheckedAt?: string | null
    class?: string
  } = $props()

  const headline = $derived(providerAvailabilityHeadline(availabilityState, availabilityReason))
  const description = $derived(
    providerAvailabilityDescription(availabilityState, availabilityReason),
  )
  const checkedAtText = $derived(providerAvailabilityCheckedAtText(availabilityCheckedAt))
</script>

<Tooltip.Root>
  <Tooltip.Trigger>
    {#snippet child({ props })}
      <Badge
        {...props}
        variant={providerAvailabilityBadgeVariant(availabilityState)}
        class={className}
      >
        {providerAvailabilityLabel(availabilityState)}
      </Badge>
    {/snippet}
  </Tooltip.Trigger>

  <Tooltip.Content
    side="top"
    sideOffset={6}
    class="bg-popover text-popover-foreground border-border block max-w-[22rem] rounded-lg border p-0 shadow-xl"
    arrowClasses="bg-popover fill-popover"
  >
    <div class="space-y-3 p-3">
      <div class="space-y-1">
        <div class="text-sm font-medium">{headline}</div>
        <p class="text-muted-foreground text-xs leading-5">{description}</p>
      </div>

      {#if availabilityReason}
        <div class="space-y-1">
          <div class="text-muted-foreground text-[11px] font-medium tracking-wide uppercase">
            Signal
          </div>
          <code class="bg-muted text-foreground inline-flex rounded px-2 py-1 text-[11px]">
            {availabilityReason}
          </code>
        </div>
      {/if}

      {#if checkedAtText}
        <div class="space-y-1">
          <div class="text-muted-foreground text-[11px] font-medium tracking-wide uppercase">
            Last Checked
          </div>
          <div class="text-xs">{checkedAtText}</div>
        </div>
      {/if}
    </div>
  </Tooltip.Content>
</Tooltip.Root>
