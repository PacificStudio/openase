<script lang="ts">
  import { Button } from '$ui/button'
  import { CircleDashed } from '@lucide/svelte'

  let { orgId }: { orgId: string } = $props()
</script>

<div class="border-border rounded-xl border border-dashed p-6 text-center">
  <CircleDashed class="text-muted-foreground mx-auto mb-2 size-8" />
  <p class="text-foreground text-sm font-medium">No providers are registered yet.</p>
  <p class="text-muted-foreground mt-1 text-xs">
    Follow the CLI guides above to install and sign in first, then register the matching provider in
    organization settings and return here to click "Recheck all providers".
  </p>
  <Button
    variant="outline"
    size="sm"
    class="mt-3"
    onclick={() => window.open(`/orgs/${orgId}/settings`, '_blank')}
  >
    Open organization settings
  </Button>
</div>
