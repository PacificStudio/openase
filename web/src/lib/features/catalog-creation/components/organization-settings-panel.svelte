<script lang="ts">
  import { invalidateAll } from '$app/navigation'
  import type { AgentProvider, Organization } from '$lib/api/contracts'
  import { ApiError } from '$lib/api/client'
  import { updateOrganization } from '$lib/api/openase'
  import { appStore } from '$lib/stores/app.svelte'
  import { Button } from '$ui/button'
  import * as Card from '$ui/card'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import * as Select from '$ui/select'

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
  let feedback = $state('')
  let error = $state('')

  function providerLabel(provider: AgentProvider) {
    return provider.available ? provider.name : `${provider.name} (Unavailable)`
  }

  function selectedProviderLabel() {
    const provider = providers.find((item) => item.id === defaultProviderId)
    return provider ? providerLabel(provider) : 'No default provider'
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
      error = 'Organization name is required.'
      feedback = ''
      return
    }
    if (!nextSlug) {
      error = 'Organization slug is required.'
      feedback = ''
      return
    }

    saving = true
    feedback = ''
    error = ''

    try {
      const payload = await updateOrganization(organization.id, {
        name: nextName,
        slug: nextSlug,
        default_agent_provider_id: defaultProviderId || null,
      })
      appStore.currentOrg = payload.organization
      feedback = 'Organization settings saved.'
      await invalidateAll()
    } catch (caughtError) {
      error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to save organization.'
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
        <Select.Trigger class="w-full">{selectedProviderLabel()}</Select.Trigger>
        <Select.Content>
          <Select.Item value="">No default provider</Select.Item>
          {#each providers as provider (provider.id)}
            <Select.Item value={provider.id}>{providerLabel(provider)}</Select.Item>
          {/each}
        </Select.Content>
      </Select.Root>
      <p class="text-muted-foreground text-xs">
        New projects can still override this, but this keeps the org-level default explicit.
      </p>
    </div>

    {#if feedback}
      <p class="text-sm text-emerald-400">{feedback}</p>
    {/if}

    {#if error}
      <p class="text-destructive text-sm">{error}</p>
    {/if}

    <div class="flex justify-end">
      <Button onclick={handleSave} disabled={saving}>
        {saving ? 'Saving…' : 'Save organization'}
      </Button>
    </div>
  </Card.Content>
</Card.Root>
