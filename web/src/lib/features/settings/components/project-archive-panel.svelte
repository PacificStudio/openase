<script lang="ts">
  import { Button } from '$ui/button'

  let {
    archiving = false,
    onArchive,
  }: {
    archiving?: boolean
    onArchive?: () => void
  } = $props()
</script>

<div class="border-destructive/30 bg-destructive/5 rounded-lg border p-4">
  <div class="space-y-2">
    <h3 class="text-foreground text-sm font-medium">Archive project</h3>
    <p class="text-muted-foreground text-sm">
      Move this project out of the active workspace surface after a confirmation step.
    </p>
  </div>

  <div class="mt-4 flex justify-start">
    <Button variant="destructive" onclick={onArchive} disabled={archiving}>
      {archiving ? 'Archiving…' : 'Archive project'}
    </Button>
  </div>
</div>
