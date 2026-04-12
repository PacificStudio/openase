<script lang="ts">
  import { goto, invalidateAll } from '$app/navigation'
  import { ApiError } from '$lib/api/client'
  import { createOrganization } from '$lib/api/openase'
  import {
    createOrganizationDraft,
    parseOrganizationDraft,
    slugFromName,
    type OrganizationCreationDraft,
  } from '$lib/features/catalog-creation/model'
  import { organizationPath } from '$lib/stores/app-context'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { Button } from '$ui/button'
  import * as Dialog from '$ui/dialog'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import { i18nStore } from '$lib/i18n/store.svelte'

  let {
    open = $bindable(false),
  }: {
    open?: boolean
  } = $props()

  let draft = $state<OrganizationCreationDraft>(createOrganizationDraft())
  let slugDirty = $state(false)
  let creating = $state(false)

  function reset() {
    draft = createOrganizationDraft()
    slugDirty = false
    creating = false
  }

  function updateName(value: string) {
    draft = {
      ...draft,
      name: value,
      slug: slugDirty ? draft.slug : slugFromName(value),
    }
  }

  function updateSlug(value: string) {
    slugDirty = true
    draft = { ...draft, slug: value }
  }

  async function handleSubmit() {
    const parsed = parseOrganizationDraft(draft)
    if (!parsed.ok) {
      toastStore.error(parsed.error)
      return
    }

    creating = true

    try {
      const payload = await createOrganization(parsed.value)
      open = false
      reset()
      await invalidateAll()
      await goto(organizationPath(payload.organization.id))
    } catch (caughtError) {
          toastStore.error(
            caughtError instanceof ApiError
              ? caughtError.detail
              : i18nStore.t('catalog.organization.dialog.errors.create'),
          )
    } finally {
      creating = false
    }
  }
</script>

<Dialog.Root
  bind:open
  onOpenChange={(next) => {
    if (!next) reset()
  }}
>
  <Dialog.Content class="sm:max-w-md">
    <Dialog.Header>
        <Dialog.Title>
          {i18nStore.t('catalog.organization.dialog.title')}
        </Dialog.Title>
      <Dialog.Description>
        {i18nStore.t('catalog.organization.dialog.description')}
      </Dialog.Description>
    </Dialog.Header>

    <form
      class="space-y-4"
      onsubmit={(event) => {
        event.preventDefault()
        void handleSubmit()
      }}
    >
      <div class="space-y-2">
        <Label for="org-name">
          {i18nStore.t('catalog.organization.dialog.labels.organizationName')}
        </Label>
        <Input
          id="org-name"
          value={draft.name}
          placeholder={i18nStore.t('catalog.organization.dialog.placeholders.organizationName')}
          oninput={(event) => updateName((event.currentTarget as HTMLInputElement).value)}
        />
      </div>

      <div class="space-y-2">
        <Label for="org-slug">
          {i18nStore.t('catalog.organization.dialog.labels.slug')}
        </Label>
        <Input
          id="org-slug"
          value={draft.slug}
          placeholder={i18nStore.t('catalog.organization.dialog.placeholders.slug')}
          oninput={(event) => updateSlug((event.currentTarget as HTMLInputElement).value)}
        />
        <p class="text-muted-foreground text-xs">
          {i18nStore.t('catalog.organization.dialog.slugHint')}
        </p>
      </div>

      <Dialog.Footer>
        <Dialog.Close>
          {#snippet child({ props })}
            <Button variant="outline" {...props}>
              {i18nStore.t('catalog.organization.dialog.actions.cancel')}
            </Button>
          {/snippet}
        </Dialog.Close>
        <Button type="submit" disabled={creating}>
          {creating
            ? i18nStore.t('catalog.organization.dialog.actions.creating')
            : i18nStore.t('catalog.organization.dialog.actions.create')}
        </Button>
      </Dialog.Footer>
    </form>
  </Dialog.Content>
</Dialog.Root>
