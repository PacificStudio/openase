<script lang="ts">
  import { tick } from 'svelte'
  import { Button } from '$ui/button'
  import * as Dialog from '$ui/dialog'
  import * as DropdownMenu from '$ui/dropdown-menu'
  import { Ellipsis, Eye, RotateCcw, TestTube2, Trash2 } from '@lucide/svelte'

  let {
    machineName,
    localMachine = false,
    resetEnabled = false,
    testing = false,
    deleting = false,
    onOpen,
    onTest,
    onReset,
    onDelete,
  }: {
    machineName: string
    localMachine?: boolean
    resetEnabled?: boolean
    testing?: boolean
    deleting?: boolean
    onOpen?: () => void
    onTest?: () => void
    onReset?: () => void
    onDelete?: () => void
  } = $props()

  let confirmResetOpen = $state(false)
  let confirmDeleteOpen = $state(false)

  async function deferMenuAction(callback?: () => void) {
    await tick()
    callback?.()
  }
</script>

<div class="flex items-center gap-1">
  <Button
    variant="ghost"
    size="icon-sm"
    title="View details"
    onclick={(event) => {
      event.stopPropagation()
      onOpen?.()
    }}
  >
    <Eye class="size-3.5" />
  </Button>

  <DropdownMenu.Root>
    <DropdownMenu.Trigger>
      {#snippet child({ props })}
        <Button
          {...props}
          variant="ghost"
          size="icon-sm"
          title="More actions"
          onclick={(event) => event.stopPropagation()}
        >
          <Ellipsis class="size-3.5" />
        </Button>
      {/snippet}
    </DropdownMenu.Trigger>
    <DropdownMenu.Content align="end" class="w-44">
      <DropdownMenu.Item
        disabled={testing}
        onclick={(event) => {
          event.stopPropagation()
          void deferMenuAction(onTest)
        }}
      >
        <TestTube2 class="size-3.5" />
        {testing ? 'Testing…' : 'Connection test'}
      </DropdownMenu.Item>
      <DropdownMenu.Item
        disabled={!resetEnabled}
        onclick={(event) => {
          event.stopPropagation()
          void deferMenuAction(() => {
            confirmResetOpen = true
          })
        }}
      >
        <RotateCcw class="size-3.5" />
        Reset draft
      </DropdownMenu.Item>
      <DropdownMenu.Separator />
      <DropdownMenu.Item
        disabled={localMachine || deleting}
        class="text-destructive data-[highlighted]:text-destructive"
        onclick={(event) => {
          event.stopPropagation()
          void deferMenuAction(() => {
            confirmDeleteOpen = true
          })
        }}
      >
        <Trash2 class="size-3.5" />
        {deleting ? 'Deleting…' : 'Delete'}
      </DropdownMenu.Item>
    </DropdownMenu.Content>
  </DropdownMenu.Root>
</div>

<Dialog.Root bind:open={confirmResetOpen}>
  <Dialog.Content class="sm:max-w-md">
    <Dialog.Header>
      <Dialog.Title>Discard unsaved changes?</Dialog.Title>
      <Dialog.Description>
        Reset will discard the unsaved edits in the open drawer for {machineName} and restore the last
        saved configuration.
      </Dialog.Description>
    </Dialog.Header>
    <Dialog.Footer class="mt-6">
      <Dialog.Close>
        {#snippet child({ props })}
          <Button variant="outline" {...props}>Cancel</Button>
        {/snippet}
      </Dialog.Close>
      <Button
        variant="destructive"
        onclick={() => {
          confirmResetOpen = false
          onReset?.()
        }}
      >
        Reset draft
      </Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>

<Dialog.Root bind:open={confirmDeleteOpen}>
  <Dialog.Content class="sm:max-w-md">
    <Dialog.Header>
      <Dialog.Title>Delete machine?</Dialog.Title>
      <Dialog.Description>
        This removes {machineName} from the organization. Existing monitor history and machine references
        may stop working.
      </Dialog.Description>
    </Dialog.Header>
    <Dialog.Footer class="mt-6">
      <Dialog.Close>
        {#snippet child({ props })}
          <Button variant="outline" {...props}>Cancel</Button>
        {/snippet}
      </Dialog.Close>
      <Button
        variant="destructive"
        disabled={deleting}
        onclick={() => {
          confirmDeleteOpen = false
          onDelete?.()
        }}
      >
        {deleting ? 'Deleting…' : 'Delete machine'}
      </Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>
