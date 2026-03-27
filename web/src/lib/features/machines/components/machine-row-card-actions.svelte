<script lang="ts">
  import { Button } from '$ui/button'
  import * as Dialog from '$ui/dialog'
  import { Pencil, RotateCcw, TestTube2, Trash2 } from '@lucide/svelte'

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
</script>

<div class="flex flex-wrap items-center justify-end gap-2 xl:flex-col xl:items-stretch">
  <Button
    size="sm"
    class="gap-1.5"
    onclick={(event) => {
      event.stopPropagation()
      onOpen?.()
    }}
  >
    <Pencil class="size-3.5" />
    Edit
  </Button>
  <Button
    size="sm"
    variant="outline"
    class="gap-1.5"
    onclick={(event) => {
      event.stopPropagation()
      onTest?.()
    }}
    disabled={testing}
  >
    <TestTube2 class="size-3.5" />
    {testing ? 'Testing…' : 'Test'}
  </Button>
  <Button
    size="sm"
    variant="outline"
    class="gap-1.5"
    onclick={(event) => {
      event.stopPropagation()
      confirmResetOpen = true
    }}
    disabled={!resetEnabled}
  >
    <RotateCcw class="size-3.5" />
    Reset
  </Button>
  <Button
    size="sm"
    variant="destructive"
    class="gap-1.5"
    onclick={(event) => {
      event.stopPropagation()
      confirmDeleteOpen = true
    }}
    disabled={localMachine || deleting}
    title={localMachine ? 'The seeded local machine cannot be deleted.' : undefined}
  >
    <Trash2 class="size-3.5" />
    {deleting ? 'Deleting…' : 'Delete'}
  </Button>
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
