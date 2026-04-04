<script lang="ts">
  import { invalidateAll } from '$app/navigation'
  import type { AgentProvider, Organization } from '$lib/api/contracts'
  import { ApiError } from '$lib/api/client'
  import { updateOrganization } from '$lib/api/openase'
  import { adapterIconPath, providerAvailabilityLabel } from '$lib/features/providers'
  import { appStore } from '$lib/stores/app.svelte'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { Button } from '$ui/button'
  import * as Card from '$ui/card'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import * as Select from '$ui/select'
  import { Wrench } from '@lucide/svelte'

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
    return provider ? providerLabel(provider) : 'No default provider'
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
      toastStore.error('Organization name is required.')
      return
    }
    if (!nextSlug) {
      toastStore.error('Organization slug is required.')
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
      toastStore.success('Organization settings saved.')
      await invalidateAll()
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to save organization.',
      )
    } finally {
      saving = false
    }
  }
</script>

<Card.Root class="rounded-2xl">
  <Card.Header>
    <Card.Title>Organization settings</Card.Title>
    <Card.Description>
      Keep the workspace label, stable slug, and default provider aligned with the current org.
    </Card.Description>
  </Card.Header>

  <Card.Content class="space-y-4">
    <div class="grid gap-4 md:grid-cols-2">
      <div class="space-y-2">
        <Label for="organization-settings-name">Organization name</Label>
        <Input id="organization-settings-name" bind:value={name} />
      </div>

      <div class="space-y-2">
        <Label for="organization-settings-slug">Slug</Label>
        <Input id="organization-settings-slug" bind:value={slug} />
      </div>
    </div>

    <div class="space-y-2">
      <Label>Default provider</Label>
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
          <Select.Item value="">No default provider</Select.Item>
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
        New projects can still override this, but this keeps the org-level default explicit.
      </p>
    </div>

    <div class="flex justify-end">
      <Button onclick={handleSave} disabled={saving}>
        {saving ? 'Saving…' : 'Save organization'}
      </Button>
    </div>
  </Card.Content>
</Card.Root>
