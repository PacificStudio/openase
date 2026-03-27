<script lang="ts">
  import { Label } from '$ui/label'
  import { Bot, Cpu, HardDrive } from '@lucide/svelte'

  let {
    providerName = 'Select an agent first',
    modelName = 'Select an agent first',
    machineSummary,
  }: {
    providerName?: string
    modelName?: string
    machineSummary: string
  } = $props()
</script>

<div class="grid gap-4 sm:grid-cols-2">
  <div class="space-y-2">
    <Label>Provider</Label>
    <div
      class="border-border text-foreground flex min-h-10 items-center gap-2 rounded-md border px-3 text-sm"
    >
      <Bot class="text-muted-foreground size-4" />
      <span>{providerName}</span>
    </div>
  </div>

  <div class="space-y-2">
    <Label>Model</Label>
    <div
      class="border-border text-foreground flex min-h-10 items-center gap-2 rounded-md border px-3 text-sm"
    >
      <Cpu class="text-muted-foreground size-4" />
      <span>{modelName}</span>
    </div>
  </div>

  <div class="space-y-2">
    <Label>Machine</Label>
    <div
      class="border-border text-foreground flex min-h-10 items-center gap-2 rounded-md border px-3 text-sm"
    >
      <HardDrive class="text-muted-foreground size-4" />
      <span>{machineSummary}</span>
    </div>
  </div>

  <div class="space-y-2">
    <Label>Workspace</Label>
    <div
      class="border-border text-foreground min-h-10 rounded-md border px-3 py-2 text-sm break-all"
    >
      OpenASE derives a fresh ticket workspace from the organization, project, and ticket.
    </div>
  </div>
</div>
