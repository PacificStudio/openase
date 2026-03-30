<script lang="ts">
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from '$ui/sheet'
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
</script>

<Sheet bind:open>
  <SheetContent
    side="right"
    class="flex w-full flex-col gap-0 p-0 sm:max-w-2xl"
    data-testid="machine-editor-sheet"
  >
    <SheetHeader class="border-border border-b px-6 py-5 text-left">
      <div class="flex items-start justify-between gap-4 pr-10">
        <div class="min-w-0 space-y-2">
          <div class="flex flex-wrap items-center gap-2">
            <SheetTitle>
              {mode === 'create' ? 'Register machine' : (machine?.name ?? 'Machine configuration')}
            </SheetTitle>
            {#if mode === 'edit' && machine}
              <Badge variant="outline" class={machineStatusBadgeClass(machine.status)}>
                {machineStatusLabel(machine.status)}
              </Badge>
            {/if}
          </div>
          <SheetDescription>
            {#if mode === 'create'}
              Create a new machine record. Runtime status is assigned by the system after monitoring
              starts.
            {:else}
              Edit SSH access and runtime configuration. System status remains read-only.
            {/if}
          </SheetDescription>
          {#if mode === 'edit' && machine}
            <p class="text-muted-foreground text-xs">{machineStatusDescription(machine.status)}</p>
          {/if}
        </div>

        <Button size="sm" onclick={onSave} disabled={saving} data-testid="machine-save-button">
          {saving ? 'Saving…' : mode === 'create' ? 'Create machine' : 'Save changes'}
        </Button>
      </div>
    </SheetHeader>

    <div class="flex-1 overflow-y-auto px-6 py-6">
      <div class="space-y-6">
        <MachineEditor {mode} {machine} {draft} {onDraftChange} />

        {#if mode === 'edit' && machine}
          <section class="border-border space-y-4 border-t pt-6">
            <div>
              <h3 class="text-foreground text-sm font-semibold">Runtime context</h3>
              <p class="text-muted-foreground mt-1 text-xs">
                Multi-level machine checks, runtime readiness, and the latest connection test
                output.
              </p>
            </div>
            <MachineHealthPanel
              {machine}
              {snapshot}
              {probe}
              loading={loadingHealth}
              refreshing={refreshingHealth}
              onRefresh={onRefreshHealth}
            />
          </section>
        {/if}
      </div>
    </div>
  </SheetContent>
</Sheet>
