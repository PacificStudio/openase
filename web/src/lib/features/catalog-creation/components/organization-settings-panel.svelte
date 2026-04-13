<script lang="ts">
  import { invalidateAll } from '$app/navigation'
  import type { AgentProvider, Organization } from '$lib/api/contracts'
  import { ApiError } from '$lib/api/client'
  import { updateOrganization } from '$lib/api/openase'
  import { adapterIconPath, providerAvailabilityLabel } from '$lib/features/providers'
  import { appStore } from '$lib/stores/app.svelte'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import * as Select from '$ui/select'
  import { Wrench } from '@lucide/svelte'
  import { i18nStore } from '$lib/i18n/store.svelte'

  let {
    organization,
    providers,
  }: {
    organization: Organization
    providers: AgentProvider[]
  } = $props()

  let name = $state('')
  let slug = $state('')
  let defaultProviderId = $state('')
  let saving = $state(false)

  function providerLabel(provider: AgentProvider) {
    const availabilityLabel = providerAvailabilityLabel(provider.availability_state)
    return `${provider.name} (${availabilityLabel})`
  }

  function selectedProviderLabel() {
    const provider = providers.find((item) => item.id === defaultProviderId)
    return provider
      ? providerLabel(provider)
      : i18nStore.t('catalog.organization.settings.labels.noDefaultProvider')
  }

  function selectedProvider() {
    return providers.find((item) => item.id === defaultProviderId) ?? null
  }

  $effect(() => {
    name = organization.name
    slug = organization.slug
    defaultProviderId = organization.default_agent_provider_id ?? ''
  })

  async function handleSave() {
    const nextName = name.trim()
    const nextSlug = slug.trim()

    if (!nextName) {
      toastStore.error(i18nStore.t('catalog.organization.settings.errors.nameRequired'))
      return
    }
    if (!nextSlug) {
      toastStore.error(i18nStore.t('catalog.organization.settings.errors.slugRequired'))
      return
    }

    saving = true

    try {
      const payload = await updateOrganization(organization.id, {
        name: nextName,
        slug: nextSlug,
        default_agent_provider_id: defaultProviderId || null,
      })
      appStore.currentOrg = payload.organization
      toastStore.success(i18nStore.t('catalog.organization.settings.success.saved'))
      await invalidateAll()
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError
          ? caughtError.detail
          : i18nStore.t('catalog.organization.settings.errors.saveFailed'),
      )
    } finally {
      saving = false
    }
  }
</script>

<div class="space-y-4">
  <div>
    <h3 class="text-foreground text-sm font-semibold">
      {i18nStore.t('catalog.organization.settings.heading')}
    </h3>
    <p class="text-muted-foreground mt-0.5 text-xs">
      {i18nStore.t('catalog.organization.settings.description')}
    </p>
  </div>

  <div class="border-border rounded-md border p-4">
    <div class="space-y-4">
      <div class="grid gap-4 md:grid-cols-2">
        <div class="space-y-1.5">
          <Label for="organization-settings-name">
            {i18nStore.t('catalog.organization.settings.labels.organizationName')}
          </Label>
          <Input id="organization-settings-name" bind:value={name} />
        </div>

        <div class="space-y-1.5">
          <Label for="organization-settings-slug">
            {i18nStore.t('catalog.organization.settings.labels.slug')}
          </Label>
          <Input id="organization-settings-slug" bind:value={slug} />
        </div>
      </div>

      <div class="space-y-1.5">
        <Label>
          {i18nStore.t('catalog.organization.settings.labels.defaultProvider')}
        </Label>
        <Select.Root
          type="single"
          value={defaultProviderId}
          onValueChange={(value) => {
            defaultProviderId = value || ''
          }}
        >
          <Select.Trigger class="h-auto w-full py-2">
            {@const provider = selectedProvider()}
            {#if provider}
              {@const iconPath = adapterIconPath(provider.adapter_type)}
              <div class="flex items-center gap-2.5">
                {#if iconPath}
                  <img src={iconPath} alt="" class="size-5 shrink-0" />
                {:else}
                  <Wrench class="text-muted-foreground size-5 shrink-0" />
                {/if}
                <div class="min-w-0 text-left">
                  <div class="text-foreground truncate text-sm font-medium">{provider.name}</div>
                  <div class="text-muted-foreground truncate text-xs">
                    {providerAvailabilityLabel(provider.availability_state)}
                  </div>
                </div>
              </div>
            {:else}
              {selectedProviderLabel()}
            {/if}
          </Select.Trigger>
          <Select.Content>
            <Select.Item value="">
              {i18nStore.t('catalog.organization.settings.labels.noDefaultProvider')}
            </Select.Item>
            {#each providers as provider (provider.id)}
              {@const iconPath = adapterIconPath(provider.adapter_type)}
              <Select.Item value={provider.id}>
                <div class="flex items-center gap-2.5 py-0.5">
                  {#if iconPath}
                    <img src={iconPath} alt="" class="size-4 shrink-0" />
                  {:else}
                    <Wrench class="text-muted-foreground size-4 shrink-0" />
                  {/if}
                  <div class="min-w-0">
                    <div class="truncate text-sm">{provider.name}</div>
                    <div class="text-muted-foreground text-xs">
                      {providerAvailabilityLabel(provider.availability_state)}
                    </div>
                  </div>
                </div>
              </Select.Item>
            {/each}
          </Select.Content>
        </Select.Root>
        <p class="text-muted-foreground text-xs">
          {i18nStore.t('catalog.organization.settings.hint.override')}
        </p>
      </div>

      <div class="flex justify-end">
        <Button onclick={handleSave} disabled={saving}>
          {saving
            ? i18nStore.t('catalog.organization.settings.actions.saving')
            : i18nStore.t('catalog.organization.settings.actions.save')}
        </Button>
      </div>
    </div>
  </div>
</div>
