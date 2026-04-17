<script lang="ts">
  import type { MachineSetupCommand } from '../machine-setup'

  let {
    commands,
    dashed = false,
    className = '',
  }: {
    commands: MachineSetupCommand[]
    dashed?: boolean
    className?: string
  } = $props()
</script>

<div class={className}>
  {#each commands as command (command.title)}
    <div
      class={`rounded-lg px-3.5 py-3 ${dashed ? 'border border-dashed' : 'border-border bg-card border'}`}
    >
      <p class="text-foreground text-sm font-medium">{command.title}</p>
      <p class="text-muted-foreground mt-1 text-xs leading-relaxed">{command.description}</p>
      <pre
        class="bg-muted/60 text-foreground mt-3 overflow-x-auto rounded-md px-3 py-2 text-xs whitespace-pre-wrap">{command.command}</pre>
    </div>
  {/each}
</div>
