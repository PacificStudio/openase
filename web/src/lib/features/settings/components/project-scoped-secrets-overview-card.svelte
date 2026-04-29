<script lang="ts">
  import { organizationPath } from '$lib/stores/app-context'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import { Textarea } from '$ui/textarea'
  import { KeyRound, Plus, X } from '@lucide/svelte'
  import { i18nStore } from '$lib/i18n/store.svelte'

  let {
    effectiveCount,
    projectOverrideCount,
    organizationSecretCount,
    organizationId,
    creating,
    onCreate,
    name = $bindable(),
    description = $bindable(),
    value = $bindable(),
  }: {
    effectiveCount: number
    projectOverrideCount: number
    organizationSecretCount: number
    organizationId: string
    creating: boolean
    onCreate: () => Promise<boolean>
    name: string
    description: string
    value: string
  } = $props()

  let formOpen = $state(false)

  // Auto-open when a pre-fill arrives from "Use as override draft"
  $effect(() => {
    if (name && !formOpen) formOpen = true
  })

  async function handleCreate() {
    const success = await onCreate()
    if (success) formOpen = false
  }

  function close() {
    formOpen = false
  }
</script>

<div class="space-y-4">
  <div class="flex flex-wrap items-center justify-between gap-3">
    <div class="flex items-center gap-2">
      <KeyRound class="text-muted-foreground size-4" />
      <h3 class="text-sm font-semibold">
        {i18nStore.t('settings.projectSecrets.overview.heading')}
      </h3>
    </div>
    <div class="flex items-center gap-3">
      {#if organizationId}
        <a
          href={`${organizationPath(organizationId)}/admin/settings`}
          class="text-muted-foreground hover:text-foreground text-sm transition-colors"
        >
          {i18nStore.t('settings.projectSecrets.overview.manageOrgLink')}
        </a>
      {/if}
      <Button size="sm" variant="outline" onclick={() => (formOpen = !formOpen)}>
        {#if formOpen}
          <X class="size-3.5" />
          {i18nStore.t('settings.projectSecrets.overview.buttons.cancel')}
        {:else}
          <Plus class="size-3.5" />
          {i18nStore.t('settings.projectSecrets.overview.buttons.newOverride')}
        {/if}
      </Button>
    </div>
  </div>

  <div
    class="bg-muted/40 flex flex-wrap items-center gap-x-4 gap-y-1 rounded-lg px-4 py-2.5 text-sm"
  >
    <span>
      <span class="font-semibold">{effectiveCount}</span>
      <span class="text-muted-foreground">
        {i18nStore.t('settings.projectSecrets.overview.stats.effective')}
      </span>
    </span>
    <span class="text-muted-foreground">·</span>
    <span>
      <span class="font-semibold">{projectOverrideCount}</span>
      <span class="text-muted-foreground">
        {i18nStore.t('settings.projectSecrets.overview.stats.projectOverrides')}
      </span>
    </span>
    <span class="text-muted-foreground">·</span>
    <span>
      <span class="font-semibold">{organizationSecretCount}</span>
      <span class="text-muted-foreground">
        {i18nStore.t('settings.projectSecrets.overview.stats.orgDefaults')}
      </span>
    </span>
  </div>

  {#if formOpen}
    <div class="border-border rounded-lg border p-4">
      <div class="mb-4">
        <h4 class="text-sm font-medium">
          {i18nStore.t('settings.projectSecrets.overview.formHeading')}
        </h4>
        <p class="text-muted-foreground mt-0.5 text-xs">
          {i18nStore.t('settings.projectSecrets.overview.formDescription')}
        </p>
      </div>

      <div class="grid gap-4 lg:grid-cols-[minmax(0,0.95fr)_minmax(0,1.05fr)]">
        <div class="space-y-3">
          <div class="space-y-1.5">
            <Label for="project-secret-name">
              {i18nStore.t('settings.projectSecrets.overview.labels.secretName')}
            </Label>
            <Input
              id="project-secret-name"
              bind:value={name}
              placeholder={i18nStore.t('settings.projectSecrets.overview.placeholders.secretName')}
            />
          </div>
          <div class="space-y-1.5">
            <Label for="project-secret-value">
              {i18nStore.t('settings.projectSecrets.overview.labels.secretValue')}
            </Label>
            <Input
              id="project-secret-value"
              type="password"
              bind:value
              placeholder={i18nStore.t('settings.projectSecrets.overview.placeholders.secretValue')}
            />
            <p class="text-muted-foreground text-xs">
              {i18nStore.t('settings.projectSecrets.overview.hints.secretValue')}
            </p>
          </div>
        </div>

        <div class="space-y-1.5">
          <Label for="project-secret-description">
            {i18nStore.t('settings.projectSecrets.overview.labels.description')}
          </Label>
          <Textarea
            id="project-secret-description"
            bind:value={description}
            rows={4}
            placeholder={i18nStore.t('settings.projectSecrets.overview.placeholders.description')}
          />
        </div>
      </div>

      <div class="mt-4 flex items-center justify-end gap-3">
        <Button variant="ghost" size="sm" onclick={close} disabled={creating}>
          {i18nStore.t('settings.projectSecrets.overview.buttons.cancel')}
        </Button>
        <Button size="sm" onclick={() => void handleCreate()} disabled={creating}>
          {creating
            ? i18nStore.t('settings.projectSecrets.overview.buttons.creating')
            : i18nStore.t('settings.projectSecrets.overview.buttons.createOverride')}
        </Button>
      </div>
    </div>
  {/if}
</div>
