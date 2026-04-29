<script lang="ts">
  import type { AgentProviderModelCatalogEntry, Machine } from '$lib/api/contracts'
  import { ProviderFormFields, type ProviderDraft } from '$lib/features/agents/public'
  import { Button } from '$ui/button'
  import { i18nStore } from '$lib/i18n/store.svelte'
  import * as Card from '$ui/card'

  let {
    draft,
    modelCatalog = [],
    machines = [],
    creating = false,
    onFieldChange,
    onAdapterChange,
    onSubmit,
  }: {
    draft: ProviderDraft
    modelCatalog?: AgentProviderModelCatalogEntry[]
    machines?: Machine[]
    creating?: boolean
    onFieldChange?: (field: keyof ProviderDraft, value: string) => void
    onAdapterChange?: (value: string) => void
    onSubmit?: () => void
  } = $props()
  const t = i18nStore.t
</script>

<Card.Root class="rounded-2xl">
  <Card.Header>
    <Card.Title>{t('catalog.providerCreation.panel.title')}</Card.Title>
    <Card.Description>{t('catalog.providerCreation.panel.description')}</Card.Description>
  </Card.Header>

  <Card.Content>
    <form
      class="space-y-4"
      onsubmit={(event) => {
        event.preventDefault()
        onSubmit?.()
      }}
    >
      <ProviderFormFields
        {draft}
        {modelCatalog}
        {machines}
        onFieldChange={(field, value) => {
          if (field === 'adapterType') {
            onAdapterChange?.(value)
          } else {
            onFieldChange?.(field, value)
          }
        }}
      />

      <Button type="submit" class="w-full" disabled={creating}>
        {creating
          ? t('catalog.providerCreation.panel.actions.creating')
          : t('catalog.providerCreation.panel.actions.create')}
      </Button>
    </form>
  </Card.Content>
</Card.Root>
