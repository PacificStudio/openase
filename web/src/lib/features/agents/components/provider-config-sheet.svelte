<script lang="ts">
  import type { AgentProviderModelCatalogEntry, Machine } from '$lib/api/contracts'
  import { listProviderModelOptions } from '$lib/api/openase'
  import { Badge } from '$ui/badge'
  import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from '$ui/sheet'
  import type { ProviderConfig, ProviderDraft, ProviderDraftField } from '../types'
  import ProviderConfigSheetEmptyState from './provider-config-sheet-empty-state.svelte'
  import ProviderConfigSheetFooter from './provider-config-sheet-footer.svelte'
  import ProviderFormFields from './provider-form-fields.svelte'

  let {
    open = $bindable(false),
    provider,
    machines,
    draft,
    saving = false,
    onDraftChange,
    onSave,
  }: {
    open?: boolean
    provider: ProviderConfig | null
    machines: Machine[]
    draft: ProviderDraft
    saving?: boolean
    onDraftChange?: (field: ProviderDraftField, value: string) => void
    onSave?: () => void
  } = $props()

  let modelCatalog = $state<AgentProviderModelCatalogEntry[]>([])

  $effect(() => {
    if (!open) {
      return
    }

    let cancelled = false
    void listProviderModelOptions()
      .then((payload) => {
        if (!cancelled) {
          modelCatalog = payload.adapter_model_options
        }
      })
      .catch(() => {
        if (!cancelled) {
          modelCatalog = []
        }
      })

    return () => {
      cancelled = true
    }
  })
</script>

<Sheet bind:open>
  <SheetContent
    side="right"
    class="flex w-full flex-col gap-0 p-0 sm:max-w-xl"
    data-testid="provider-config-sheet"
  >
    <SheetHeader class="border-border border-b px-4 py-4 text-left sm:px-6 sm:py-5">
      <div class="flex items-center gap-2">
        <SheetTitle>{provider?.name ?? 'Provider configuration'}</SheetTitle>
        {#if provider?.isDefault}
          <Badge variant="outline" class="text-[10px]">Default</Badge>
        {/if}
      </div>
      <SheetDescription>
        Update adapter wiring, CLI launch settings, model tuning, and token costs.
      </SheetDescription>
    </SheetHeader>

    {#if provider}
      <div class="flex-1 overflow-y-auto px-4 py-4 sm:px-6 sm:py-5">
        <ProviderFormFields
          {draft}
          {modelCatalog}
          {machines}
          showCostFields
          onFieldChange={onDraftChange}
        />
      </div>

      <ProviderConfigSheetFooter {saving} onCancel={() => (open = false)} {onSave} />
    {:else}
      <ProviderConfigSheetEmptyState />
    {/if}
  </SheetContent>
</Sheet>
