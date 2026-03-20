<script lang="ts">
  import { appStore } from '$lib/stores/app.svelte'
  import { listStatuses } from '$lib/api/openase'
  import { ApiError } from '$lib/api/client'
  import { Input } from '$ui/input'
  import { Separator } from '$ui/separator'
  type StatusItem = {
    id: string
    name: string
    color: string
  }

  let statuses = $state<StatusItem[]>([])
  let loading = $state(false)
  let error = $state('')

  $effect(() => {
    const projectId = appStore.currentProject?.id
    if (!projectId) {
      statuses = []
      return
    }

    let cancelled = false

    const load = async () => {
      loading = true
      error = ''

      try {
        const payload = await listStatuses(projectId)
        if (cancelled) return
        statuses = payload.statuses
          .slice()
          .sort((left, right) => left.position - right.position)
          .map((status) => ({
            id: status.id,
            name: status.name,
            color: status.color,
          }))
      } catch (caughtError) {
        if (cancelled) return
        error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to load statuses.'
      } finally {
        if (!cancelled) {
          loading = false
        }
      }
    }

    void load()

    return () => {
      cancelled = true
    }
  })
</script>

<div class="max-w-lg space-y-6">
  <div>
    <h2 class="text-foreground text-base font-semibold">Statuses</h2>
    <p class="text-muted-foreground mt-1 text-sm">
      Current board columns. Status mutations are not exposed by the current API.
    </p>
  </div>

  <Separator />

  {#if loading}
    <div class="text-muted-foreground text-sm">Loading statuses…</div>
  {:else if error}
    <div class="text-destructive text-sm">{error}</div>
  {:else}
    <div class="space-y-2">
      {#each statuses as status (status.id)}
        <div class="border-border flex items-center gap-3 rounded-md border px-3 py-2">
          <input
            type="color"
            value={status.color}
            disabled
            class="size-6 shrink-0 cursor-not-allowed rounded border-0 bg-transparent p-0 opacity-70"
          />
          <Input value={status.name} disabled class="h-8 flex-1 text-sm" />
        </div>
      {/each}
    </div>
    <p class="text-muted-foreground text-xs">
      Add, delete, and reorder actions stay disabled until status mutation endpoints are available.
    </p>
  {/if}
</div>
