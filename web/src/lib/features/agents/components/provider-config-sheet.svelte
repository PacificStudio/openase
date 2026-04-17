<script lang="ts">
  import type { AgentProviderModelCatalogEntry, Machine } from '$lib/api/contracts'
  import { i18nStore } from '$lib/i18n/store.svelte'
  import { listProviderModelOptions } from '$lib/api/openase'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import * as Dialog from '$ui/dialog'
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
    deleting = false,
    onDraftChange,
    onSave,
    onDelete,
  }: {
    open?: boolean
    provider: ProviderConfig | null
    machines: Machine[]
    draft: ProviderDraft
    saving?: boolean
    deleting?: boolean
    onDraftChange?: (field: ProviderDraftField, value: string) => void
    onSave?: () => void
    onDelete?: () => void
  } = $props()

  let modelCatalog = $state<AgentProviderModelCatalogEntry[]>([])
  let deleteDialogOpen = $state(false)

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
          <Badge variant="outline" class="text-[10px]">
            {i18nStore.t('agents.providerConfig.defaultBadge')}
          </Badge>
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

      <ProviderConfigSheetFooter
        {saving}
        {deleting}
        onCancel={() => (open = false)}
        {onSave}
        onDelete={() => (deleteDialogOpen = true)}
      />
    {:else}
      <ProviderConfigSheetEmptyState />
    {/if}
  </SheetContent>
</Sheet>

<Dialog.Root bind:open={deleteDialogOpen}>
  <Dialog.Content class="sm:max-w-md">
    <Dialog.Header>
      <Dialog.Title>Delete provider?</Dialog.Title>
      <Dialog.Description>
        {provider
          ? `This permanently deletes ${provider.name}. Deletion is blocked if the provider is still referenced by defaults, agents, or runtime resources.`
          : 'This permanently deletes the selected provider.'}
      </Dialog.Description>
    </Dialog.Header>
    <Dialog.Footer class="mt-6">
      <Dialog.Close>
        {#snippet child({ props })}
          <Button variant="outline" {...props} disabled={deleting}>
            {i18nStore.t('common.cancel')}
          </Button>
        {/snippet}
      </Dialog.Close>
      <Button variant="destructive" onclick={() => onDelete?.()} disabled={deleting}>
        {deleting ? 'Deleting…' : 'Delete provider'}
      </Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>
