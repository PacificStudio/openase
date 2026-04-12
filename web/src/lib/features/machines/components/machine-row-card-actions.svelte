<script lang="ts">
  import { tick } from 'svelte'
  import { Button } from '$ui/button'
  import * as Dialog from '$ui/dialog'
  import * as DropdownMenu from '$ui/dropdown-menu'
  import { Ellipsis, Eye, RotateCcw, TestTube2, Trash2 } from '@lucide/svelte'
  import { i18nStore } from '$lib/i18n/store.svelte'

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
      title={i18nStore.t('machines.machineRowCardActions.action.viewDetails')}
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
            title={i18nStore.t('machines.machineRowCardActions.action.moreActions')}
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
        {testing
          ? i18nStore.t('machines.machineRowCardActions.action.testing')
          : i18nStore.t('machines.machineRowCardActions.action.connectionTest')}
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
        {i18nStore.t('machines.machineRowCardActions.action.resetDraft')}
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
        {deleting
          ? i18nStore.t('machines.machineRowCardActions.action.deleting')
          : i18nStore.t('machines.machineRowCardActions.action.delete')}
      </DropdownMenu.Item>
    </DropdownMenu.Content>
  </DropdownMenu.Root>
</div>

<Dialog.Root bind:open={confirmResetOpen}>
  <Dialog.Content class="sm:max-w-md">
    <Dialog.Header>
      <Dialog.Title>
        {i18nStore.t('machines.machineRowCardActions.dialog.reset.title')}
      </Dialog.Title>
      <Dialog.Description>
        {i18nStore.t('machines.machineRowCardActions.dialog.reset.description', { machineName })}
      </Dialog.Description>
    </Dialog.Header>
    <Dialog.Footer class="mt-6">
      <Dialog.Close>
          {#snippet child({ props })}
            <Button variant="outline" {...props}>{i18nStore.t('common.cancel')}</Button>
          {/snippet}
      </Dialog.Close>
        <Button
          variant="destructive"
          onclick={() => {
            confirmResetOpen = false
            onReset?.()
          }}
        >
          {i18nStore.t('machines.machineRowCardActions.action.resetDraft')}
        </Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>

<Dialog.Root bind:open={confirmDeleteOpen}>
  <Dialog.Content class="sm:max-w-md">
    <Dialog.Header>
      <Dialog.Title>
        {i18nStore.t('machines.machineRowCardActions.dialog.delete.title')}
      </Dialog.Title>
      <Dialog.Description>
        {i18nStore.t('machines.machineRowCardActions.dialog.delete.description', { machineName })}
      </Dialog.Description>
    </Dialog.Header>
    <Dialog.Footer class="mt-6">
      <Dialog.Close>
          {#snippet child({ props })}
            <Button variant="outline" {...props}>{i18nStore.t('common.cancel')}</Button>
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
        {deleting
          ? i18nStore.t('machines.machineRowCardActions.action.deleting')
          : i18nStore.t('machines.machineRowCardActions.action.deleteMachine')}
      </Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>
