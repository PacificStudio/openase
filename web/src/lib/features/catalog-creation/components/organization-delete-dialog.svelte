<script lang="ts">
  import { invalidateAll } from '$app/navigation'
  import type { Organization } from '$lib/api/contracts'
  import { ApiError } from '$lib/api/client'
  import { archiveOrganization } from '$lib/api/openase'
  import { i18nStore } from '$lib/i18n/store.svelte'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { Button } from '$ui/button'
  import * as Dialog from '$ui/dialog'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'

  let {
    organization,
    open = $bindable(false),
  }: {
    organization: Organization
    open?: boolean
  } = $props()

  let confirmation = $state('')
  let archiving = $state(false)

  const t = i18nStore.t

  const confirmed = $derived(confirmation === organization.name)

  function reset() {
    confirmation = ''
    archiving = false
  }

  async function handleArchive() {
    if (!confirmed) return

    archiving = true

    try {
      await archiveOrganization(organization.id)
      open = false
      reset()
      await invalidateAll()
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError && 'detail' in caughtError && caughtError.detail
          ? caughtError.detail
          : t('catalog.organization.delete.errors.generic'),
      )
    } finally {
      archiving = false
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
      <Dialog.Title>{t('catalog.organization.delete.title')}</Dialog.Title>
      <Dialog.Description>
        {t('catalog.organization.delete.description', { name: organization.name })}
      </Dialog.Description>
    </Dialog.Header>

    <form
      class="space-y-4"
      onsubmit={(event) => {
        event.preventDefault()
        void handleArchive()
      }}
    >
      <div class="space-y-2">
        <Label for="delete-confirmation">
          {t('catalog.organization.delete.confirmLabel', { name: organization.name })}
        </Label>
        <Input
          id="delete-confirmation"
          value={confirmation}
          placeholder={organization.name}
          oninput={(event) => {
            confirmation = (event.currentTarget as HTMLInputElement).value
          }}
        />
      </div>

      <Dialog.Footer>
        <Dialog.Close>
          {#snippet child({ props })}
            <Button variant="outline" {...props}>
              {t('catalog.organization.delete.actions.cancel')}
            </Button>
          {/snippet}
        </Dialog.Close>
        <Button variant="destructive" type="submit" disabled={!confirmed || archiving}>
          {archiving
            ? t('catalog.organization.delete.actions.archiving')
            : t('catalog.organization.delete.actions.archive')}
        </Button>
      </Dialog.Footer>
    </form>
  </Dialog.Content>
</Dialog.Root>
