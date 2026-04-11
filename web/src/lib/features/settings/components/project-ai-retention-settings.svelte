<script lang="ts">
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import { Switch } from '$ui/switch'

  let {
    enabled = $bindable(false),
    keepLatestN = $bindable(''),
    keepRecentDays = $bindable(''),
  }: {
    enabled?: boolean
    keepLatestN?: string
    keepRecentDays?: string
  } = $props()
</script>

<section class="space-y-3">
  <div class="flex items-start justify-between gap-4">
    <div class="min-w-0 space-y-1">
      <h3 class="text-foreground text-sm font-semibold">Project AI retention</h3>
      <p class="text-muted-foreground text-xs leading-relaxed">
        Retain a conversation if it is within the latest N conversations or active within the last M
        days.
      </p>
    </div>
    <Switch
      checked={enabled}
      onCheckedChange={(value) => (enabled = value)}
      aria-label="Enable Project AI retention"
    />
  </div>

  {#if enabled}
    <div class="border-border/60 bg-muted/30 space-y-4 rounded-lg border p-4">
      <div class="grid gap-4 sm:grid-cols-2">
        <div class="space-y-1.5">
          <Label for="keep-latest-conversations" class="text-xs font-medium">
            Keep latest conversations
          </Label>
          <Input
            id="keep-latest-conversations"
            type="number"
            min="0"
            step="1"
            value={keepLatestN}
            oninput={(event) => {
              keepLatestN = (event.currentTarget as HTMLInputElement).value
            }}
            class="h-8 w-28"
            placeholder="0"
          />
          <p class="text-muted-foreground text-[11px] leading-relaxed">
            Keep the latest N conversations per user in this project.
          </p>
        </div>

        <div class="space-y-1.5">
          <Label for="keep-recent-days" class="text-xs font-medium">Keep recent days</Label>
          <Input
            id="keep-recent-days"
            type="number"
            min="0"
            step="1"
            value={keepRecentDays}
            oninput={(event) => {
              keepRecentDays = (event.currentTarget as HTMLInputElement).value
            }}
            class="h-8 w-28"
            placeholder="0"
          />
          <p class="text-muted-foreground text-[11px] leading-relaxed">
            Keep conversations with activity in the last M days.
          </p>
        </div>
      </div>

      <p class="text-muted-foreground border-border/50 border-t pt-3 text-[11px] leading-relaxed">
        Auto-prune skips dirty workspaces by default and preserves live runtimes plus pending user
        interrupts.
      </p>
    </div>
  {/if}
</section>
