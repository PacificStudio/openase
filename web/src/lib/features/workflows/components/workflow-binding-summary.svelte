<script lang="ts">
  import { Label } from '$ui/label'
  import { Bot, Cpu, HardDrive } from '@lucide/svelte'
  import { t } from './i18n'

  let {
    providerName = t('workflows.bindingSummary.placeholder.selectAgent'),
    modelName = t('workflows.bindingSummary.placeholder.selectAgent'),
    machineSummary,
  }: {
    providerName?: string
    modelName?: string
    machineSummary: string
  } = $props()
</script>

<div class="grid gap-4 sm:grid-cols-2">
  <div class="space-y-2">
    <Label>
      {t('workflows.bindingSummary.labels.provider')}
    </Label>
    <div
      class="border-border text-foreground flex min-h-10 items-center gap-2 rounded-md border px-3 text-sm"
    >
      <Bot class="text-muted-foreground size-4" />
      <span>{providerName}</span>
    </div>
  </div>

  <div class="space-y-2">
    <Label>
      {t('workflows.bindingSummary.labels.model')}
    </Label>
    <div
      class="border-border text-foreground flex min-h-10 items-center gap-2 rounded-md border px-3 text-sm"
    >
      <Cpu class="text-muted-foreground size-4" />
      <span>{modelName}</span>
    </div>
  </div>

  <div class="space-y-2">
    <Label>
      {t('workflows.bindingSummary.labels.machine')}
    </Label>
    <div
      class="border-border text-foreground flex min-h-10 items-center gap-2 rounded-md border px-3 text-sm"
    >
      <HardDrive class="text-muted-foreground size-4" />
      <span>{machineSummary}</span>
    </div>
  </div>

  <div class="space-y-2">
    <Label>
      {t('workflows.bindingSummary.labels.workspace')}
    </Label>
    <div
      class="border-border text-foreground min-h-10 rounded-md border px-3 py-2 text-sm break-all"
    >
      {t('workflows.bindingSummary.workspaceDescription')}
    </div>
  </div>
</div>
