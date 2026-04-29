<script lang="ts">
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Sheet, SheetContent, SheetHeader, SheetTitle } from '$ui/sheet'
  import { Textarea } from '$ui/textarea'
  import { i18nStore } from '$lib/i18n/store.svelte'

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
        <SheetTitle class="text-base">{i18nStore.t('skills.createSheet.title')}</SheetTitle>
        <Button size="sm" onclick={onCreate} disabled={creating}>
          {creating
            ? i18nStore.t('skills.createSheet.actions.creating')
            : i18nStore.t('skills.createSheet.actions.create')}
        </Button>
      </div>
    </SheetHeader>

    <div class="flex-1 space-y-4 overflow-y-auto px-6 py-5">
      <div class="grid gap-4 sm:grid-cols-2">
        <div class="space-y-1.5">
          <span class="text-muted-foreground text-[11px] font-medium tracking-wider uppercase">
            {i18nStore.t('skills.createSheet.labels.name')}
          </span>
          <Input
            bind:value={name}
            placeholder={i18nStore.t('skills.createSheet.placeholders.name')}
            class="h-9 text-sm"
            disabled={creating}
          />
        </div>
        <div class="space-y-1.5">
          <span class="text-muted-foreground text-[11px] font-medium tracking-wider uppercase">
            {i18nStore.t('skills.createSheet.labels.description')}
          </span>
          <Input
            bind:value={description}
            placeholder={i18nStore.t('skills.createSheet.placeholders.description')}
            class="h-9 text-sm"
            disabled={creating}
          />
        </div>
      </div>

      <div class="space-y-1.5">
        <span class="text-muted-foreground text-[11px] font-medium tracking-wider uppercase">
          {i18nStore.t('skills.createSheet.labels.skillMd')}
        </span>
        <Textarea bind:value={content} class="min-h-64 font-mono text-sm" disabled={creating} />
      </div>

      <label class="flex items-center gap-2 text-sm">
        <input bind:checked={enabled} type="checkbox" disabled={creating} />
        {i18nStore.t('skills.createSheet.labels.enableImmediately')}
      </label>
    </div>
  </SheetContent>
</Sheet>
