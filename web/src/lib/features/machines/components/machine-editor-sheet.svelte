<script lang="ts">
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from '$ui/sheet'
  import * as Tabs from '$ui/tabs'
  import MachineEditor from './machine-editor.svelte'
  import MachineHealthPanel from './machine-health-panel.svelte'
  import { machineStatusBadgeClass, machineStatusDescription, machineStatusLabel } from '../model'
  import type {
    MachineDraft,
    MachineDraftField,
    MachineEditorMode,
    MachineItem,
    MachineProbeResult,
    MachineSnapshot,
  } from '../types'

  let {
    open = $bindable(false),
    mode,
    machine,
    draft,
    snapshot,
    probe,
    loadingHealth = false,
    refreshingHealth = false,
    saving = false,
    onDraftChange,
    onRefreshHealth,
    onSave,
  }: {
    open?: boolean
    mode: MachineEditorMode
    machine: MachineItem | null
    draft: MachineDraft
    snapshot: MachineSnapshot | null
    probe: MachineProbeResult | null
    loadingHealth?: boolean
    refreshingHealth?: boolean
    saving?: boolean
    onDraftChange?: (field: MachineDraftField, value: string) => void
    onRefreshHealth?: () => void
    onSave?: () => void
  } = $props()

  let activeTab = $state('configuration')
</script>

<Sheet bind:open>
  <SheetContent
    side="right"
    class="flex w-full flex-col gap-0 p-0 sm:max-w-2xl"
    data-testid="machine-editor-sheet"
  >
    <SheetHeader class="border-border space-y-0 border-b px-4 py-3 text-left sm:px-6 sm:py-4">
      <div class="flex items-start justify-between gap-3 pr-8 sm:gap-4 sm:pr-10">
        <div class="min-w-0 space-y-1.5">
          <div class="flex flex-wrap items-center gap-2">
            <SheetTitle class="text-sm sm:text-base">
              {mode === 'create' ? 'Register machine' : (machine?.name ?? 'Machine')}
            </SheetTitle>
            {#if mode === 'edit' && machine}
              <Badge variant="outline" class={machineStatusBadgeClass(machine.status)}>
                {machineStatusLabel(machine.status)}
              </Badge>
            {/if}
          </div>
          <SheetDescription class="text-xs">
            {#if mode === 'create'}
              Define identity, topology, helper access, and workspace for a new machine.
            {:else}
              {machineStatusDescription(machine?.status ?? '')}
            {/if}
          </SheetDescription>
        </div>

        <Button size="sm" onclick={onSave} disabled={saving} data-testid="machine-save-button">
          {saving ? 'Saving...' : mode === 'create' ? 'Create' : 'Save'}
        </Button>
      </div>

      {#if mode === 'edit' && machine}
        <div class="pt-3">
          <Tabs.Root bind:value={activeTab}>
            <Tabs.List variant="line">
              <Tabs.Trigger value="configuration">Configuration</Tabs.Trigger>
              <Tabs.Trigger value="health">Health, Setup & Status</Tabs.Trigger>
            </Tabs.List>
          </Tabs.Root>
        </div>
      {/if}
    </SheetHeader>

    <div class="flex-1 overflow-y-auto px-4 py-4 sm:px-6 sm:py-5">
      {#if mode === 'create' || activeTab === 'configuration'}
        <MachineEditor {machine} {draft} {onDraftChange} />
      {:else if activeTab === 'health' && machine}
        <MachineHealthPanel
          {machine}
          {snapshot}
          {probe}
          loading={loadingHealth}
          refreshing={refreshingHealth}
          onRefresh={onRefreshHealth}
        />
      {/if}
    </div>
  </SheetContent>
</Sheet>
