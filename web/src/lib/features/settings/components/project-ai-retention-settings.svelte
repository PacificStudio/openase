<script lang="ts">
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import { Switch } from '$ui/switch'
  import { i18nStore } from '$lib/i18n/store.svelte'

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
      <h3 class="text-foreground text-sm font-semibold">
        {i18nStore.t('settings.retention.heading')}
      </h3>
      <p class="text-muted-foreground text-xs leading-relaxed">
        {i18nStore.t('settings.retention.description')}
      </p>
    </div>
    <Switch
      checked={enabled}
      onCheckedChange={(value) => (enabled = value)}
      aria-label={i18nStore.t('settings.retention.toggleAriaLabel')}
    />
  </div>

  {#if enabled}
    <div class="border-border/60 bg-muted/30 space-y-4 rounded-lg border p-4">
      <div class="grid gap-4 sm:grid-cols-2">
        <div class="space-y-1.5">
          <Label for="keep-latest-conversations" class="text-xs font-medium">
            {i18nStore.t('settings.retention.labels.keepLatest')}
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
            placeholder={i18nStore.t('settings.retention.placeholders.keepLatest')}
          />
          <p class="text-muted-foreground text-[11px] leading-relaxed">
            {i18nStore.t('settings.retention.hints.keepLatest')}
          </p>
        </div>

        <div class="space-y-1.5">
          <Label for="keep-recent-days" class="text-xs font-medium">
            {i18nStore.t('settings.retention.labels.keepRecent')}
          </Label>
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
            placeholder={i18nStore.t('settings.retention.placeholders.keepRecent')}
          />
          <p class="text-muted-foreground text-[11px] leading-relaxed">
            {i18nStore.t('settings.retention.hints.keepRecent')}
          </p>
        </div>
      </div>

      <p class="text-muted-foreground border-border/50 border-t pt-3 text-[11px] leading-relaxed">
        {i18nStore.t('settings.retention.hints.autoPrune')}
      </p>
    </div>
  {/if}
</section>
