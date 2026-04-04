<script lang="ts">
  type SelectedAgentBinding = {
    providerName: string
    modelName: string
    machineName: string
  }

  let {
    selectedAgent,
  }: {
    selectedAgent: SelectedAgentBinding | null
  } = $props()
</script>

{#if selectedAgent}
  <div class="bg-muted/40 border-border/70 space-y-1.5 rounded-lg border px-3 py-2.5">
    <div class="flex items-center justify-between text-xs">
      <span class="text-muted-foreground">Provider</span>
      <span class="text-foreground font-medium">{selectedAgent.providerName}</span>
    </div>
    <div class="flex items-center justify-between text-xs">
      <span class="text-muted-foreground">Model</span>
      <span class="text-foreground font-mono">{selectedAgent.modelName}</span>
    </div>
    <div class="flex items-center justify-between text-xs">
      <span class="text-muted-foreground">Machine</span>
      <span class="text-foreground">{selectedAgent.machineName}</span>
    </div>
  </div>
{/if}
