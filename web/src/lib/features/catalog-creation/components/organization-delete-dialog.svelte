<script lang="ts">
  import { invalidateAll } from '$app/navigation'
  import type { Organization } from '$lib/api/contracts'
  import { ApiError } from '$lib/api/client'
  import { archiveOrganization } from '$lib/api/openase'
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
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to archive organization.',
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
      <Dialog.Title>Archive organization</Dialog.Title>
      <Dialog.Description>
        This will archive <strong>{organization.name}</strong>, automatically archive every project
        in it, and remove the organization from active navigation.
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
          Type <strong class="text-foreground">{organization.name}</strong> to confirm
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
            <Button variant="outline" {...props}>Cancel</Button>
          {/snippet}
        </Dialog.Close>
        <Button variant="destructive" type="submit" disabled={!confirmed || archiving}>
          {archiving ? 'Archiving...' : 'Archive organization'}
        </Button>
      </Dialog.Footer>
    </form>
  </Dialog.Content>
</Dialog.Root>
