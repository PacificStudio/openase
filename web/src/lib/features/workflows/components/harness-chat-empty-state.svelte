<script lang="ts">
  import { WandSparkles } from '@lucide/svelte'
</script>

<div class="border-border bg-muted/30 rounded-xl border border-dashed px-4 py-4 text-sm">
  <div class="flex items-center gap-2 font-medium">
    <WandSparkles class="text-primary size-4" />
    Ask for a harness rewrite, guardrail tweak, or a new workflow handoff.
  </div>
  <p class="text-muted-foreground mt-2 text-xs leading-5">
    When the assistant returns a full harness draft, you can preview the diff here and apply it into
    the editor without leaving the page.
  </p>
</div>
