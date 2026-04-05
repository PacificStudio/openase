<script lang="ts">
  import { Button } from '$ui/button'
  import { CircleDashed } from '@lucide/svelte'

  let { orgId }: { orgId: string } = $props()
</script>

<div class="border-border rounded-lg border border-dashed px-6 py-8 text-center">
  <CircleDashed class="text-muted-foreground mx-auto mb-2 size-6" />
  <p class="text-foreground text-sm font-medium">No providers registered yet</p>
  <p class="text-muted-foreground mx-auto mt-1 max-w-sm text-xs">
    Follow a CLI guide above, then register the provider in organization settings.
  </p>
  <Button
    variant="outline"
    size="sm"
    class="mt-4"
    onclick={() => window.open(`/orgs/${orgId}/settings`, '_blank')}
  >
    Organization settings
  </Button>
</div>
