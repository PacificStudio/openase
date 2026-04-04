<script lang="ts">
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Sheet, SheetContent, SheetHeader, SheetTitle } from '$ui/sheet'
  import { Textarea } from '$ui/textarea'

  let {
    open = $bindable(false),
    name = $bindable(''),
    description = $bindable(''),
    content = $bindable(''),
    enabled = $bindable(true),
    creating = false,
    onCreate,
  }: {
    open?: boolean
    name?: string
    description?: string
    content?: string
    enabled?: boolean
    creating?: boolean
    onCreate?: () => void
  } = $props()
</script>

<Sheet bind:open>
  <SheetContent side="right" class="flex w-full flex-col gap-0 p-0 sm:max-w-xl">
    <SheetHeader class="border-border shrink-0 border-b px-6 py-4 text-left">
      <div class="flex items-center justify-between gap-4 pr-10">
        <SheetTitle class="text-base">New Skill</SheetTitle>
        <Button size="sm" onclick={onCreate} disabled={creating}>
          {creating ? 'Creating…' : 'Create'}
        </Button>
      </div>
    </SheetHeader>

    <div class="flex-1 space-y-4 overflow-y-auto px-6 py-5">
      <div class="grid gap-4 sm:grid-cols-2">
        <div class="space-y-1.5">
          <span class="text-muted-foreground text-[11px] font-medium tracking-wider uppercase">
            Name
          </span>
          <Input
            bind:value={name}
            placeholder="deploy-docker"
            class="h-9 text-sm"
            disabled={creating}
          />
        </div>
        <div class="space-y-1.5">
          <span class="text-muted-foreground text-[11px] font-medium tracking-wider uppercase">
            Description
          </span>
          <Input
            bind:value={description}
            placeholder="Human-readable description"
            class="h-9 text-sm"
            disabled={creating}
          />
        </div>
      </div>

      <div class="space-y-1.5">
        <span class="text-muted-foreground text-[11px] font-medium tracking-wider uppercase">
          SKILL.md
        </span>
        <Textarea bind:value={content} class="min-h-64 font-mono text-sm" disabled={creating} />
      </div>

      <label class="flex items-center gap-2 text-sm">
        <input bind:checked={enabled} type="checkbox" disabled={creating} />
        Enable immediately
      </label>
    </div>
  </SheetContent>
</Sheet>
