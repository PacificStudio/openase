<script lang="ts">
  import { WandSparkles } from '@lucide/svelte'
  import { i18nStore } from '$lib/i18n/store.svelte'
</script>

<div class="text-muted-foreground flex items-center gap-1.5 px-1 py-1 text-xs">
  <WandSparkles class="text-primary size-3 shrink-0" />
  <span>{i18nStore.t('skills.chatEmptyState.message')}</span>
</div>
