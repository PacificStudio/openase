<script lang="ts">
  import { invalidateAll } from '$app/navigation'
  import type { Organization } from '$lib/api/contracts'
  import { ApiError } from '$lib/api/client'
  import { deleteOrganization } from '$lib/api/openase'
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
  let deleting = $state(false)
  let error = $state('')

  const confirmed = $derived(confirmation === organization.name)

  function reset() {
    confirmation = ''
    deleting = false
    error = ''
  }

  async function handleDelete() {
    if (!confirmed) return

    deleting = true
    error = ''

    try {
      await deleteOrganization(organization.id)
      open = false
      reset()
      await invalidateAll()
    } catch (caughtError) {
      error =
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to delete organization.'
    } finally {
      deleting = false
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
      <Dialog.Title>Delete organization</Dialog.Title>
      <Dialog.Description>
        This will permanently delete <strong>{organization.name}</strong> and all its providers and
        machines. Active projects must be archived or deleted first.
      </Dialog.Description>
    </Dialog.Header>

    <form
      class="space-y-4"
      onsubmit={(event) => {
        event.preventDefault()
        void handleDelete()
      }}
    >
      <div class="space-y-2">
        <Label for="delete-confirmation">
          Type <strong class="text-foreground">{organization.name}</strong> to confirm
        </Label>
        <Input
          id="delete-confirmation"
          value={confirmation}
          placeholder={organization.name}
          oninput={(event) => {
            confirmation = (event.currentTarget as HTMLInputElement).value
            error = ''
          }}
        />
      </div>

      {#if error}
        <p class="text-destructive text-sm">{error}</p>
      {/if}

      <Dialog.Footer>
        <Dialog.Close>
          {#snippet child({ props })}
            <Button variant="outline" {...props}>Cancel</Button>
          {/snippet}
        </Dialog.Close>
        <Button variant="destructive" type="submit" disabled={!confirmed || deleting}>
          {deleting ? 'Deleting...' : 'Delete organization'}
        </Button>
      </Dialog.Footer>
    </form>
  </Dialog.Content>
</Dialog.Root>
