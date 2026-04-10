<script lang="ts">
  import { Checkbox } from '$ui/checkbox'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'

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

<div class="space-y-3 rounded-lg border p-4">
  <div class="space-y-1">
    <h3 class="text-sm font-medium">Project AI retention</h3>
    <p class="text-muted-foreground text-xs">
      Retain a conversation if it is within the latest N conversations or active within the last M
      days.
    </p>
    <p class="text-muted-foreground text-xs">
      Auto-prune skips dirty workspaces by default and preserves live runtimes plus pending user
      interrupts.
    </p>
  </div>

  <div class="flex items-center gap-2">
    <Checkbox id="project-ai-retention-enabled" bind:checked={enabled} />
    <Label for="project-ai-retention-enabled" class="text-sm font-medium">
      Enable Project AI retention
    </Label>
  </div>

  <div class="grid gap-4 md:grid-cols-2">
    <div class="space-y-2">
      <Label for="keep-latest-conversations">Keep latest conversations</Label>
      <Input
        id="keep-latest-conversations"
        type="number"
        min="0"
        step="1"
        value={keepLatestN}
        oninput={(event) => {
          keepLatestN = (event.currentTarget as HTMLInputElement).value
        }}
        class="w-32"
        placeholder="0"
      />
      <p class="text-muted-foreground text-xs">
        Keep the latest N conversations per user in this project.
      </p>
    </div>

    <div class="space-y-2">
      <Label for="keep-recent-days">Keep recent days</Label>
      <Input
        id="keep-recent-days"
        type="number"
        min="0"
        step="1"
        value={keepRecentDays}
        oninput={(event) => {
          keepRecentDays = (event.currentTarget as HTMLInputElement).value
        }}
        class="w-32"
        placeholder="0"
      />
      <p class="text-muted-foreground text-xs">
        Keep conversations with activity in the last M days.
      </p>
    </div>
  </div>
</div>
