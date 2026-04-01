<script lang="ts">
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import ProviderPricingSummary from './provider-pricing-summary.svelte'

  let {
    costPerInputToken,
    costPerOutputToken,
    routedOfficialPricing = false,
    pricingStatus = '',
    pricingRows = [],
    pricingNotes = [],
    onPricingFieldChange,
  }: {
    costPerInputToken: string
    costPerOutputToken: string
    routedOfficialPricing?: boolean
    pricingStatus?: string
    pricingRows?: [string, string][]
    pricingNotes?: string[]
    onPricingFieldChange?: (
      field: 'costPerInputToken' | 'costPerOutputToken',
      value: string,
    ) => void
  } = $props()

  function fieldValue(event: Event) {
    return (event.currentTarget as HTMLInputElement).value
  }
</script>

<div class="space-y-2">
  <Label for="provider-cost-input">Input pricing (USD / 1M tokens)</Label>
  <Input
    id="provider-cost-input"
    type="number"
    min="0"
    step="0.01"
    placeholder="3.00"
    value={costPerInputToken}
    disabled={routedOfficialPricing}
    oninput={(event) => onPricingFieldChange?.('costPerInputToken', fieldValue(event))}
  />
  <p class="text-muted-foreground text-xs">
    Enter the published per-million-token rate. Example: `$3.00 / 1M` stores `0.000003` USD per
    token internally.
  </p>
</div>

<div class="space-y-2">
  <Label for="provider-cost-output">Output pricing (USD / 1M tokens)</Label>
  <Input
    id="provider-cost-output"
    type="number"
    min="0"
    step="0.01"
    placeholder="15.00"
    value={costPerOutputToken}
    disabled={routedOfficialPricing}
    oninput={(event) => onPricingFieldChange?.('costPerOutputToken', fieldValue(event))}
  />
  <p class="text-muted-foreground text-xs">
    Use provider list pricing as-is here, in `USD / 1M tokens`, to avoid 1,000x or 1,000,000x entry
    mistakes.
  </p>
</div>

<ProviderPricingSummary status={pricingStatus} rows={pricingRows} notes={pricingNotes} />
