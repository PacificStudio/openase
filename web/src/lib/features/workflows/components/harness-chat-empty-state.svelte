<script lang="ts">
  import { WandSparkles } from '@lucide/svelte'
  import { t } from './i18n'
</script>

<div class="text-muted-foreground flex items-center gap-1.5 px-1 py-1 text-xs">
  <WandSparkles class="text-primary size-3 shrink-0" />
  <span>
    {t('workflows.harness.chatEmptyState')}
  </span>
</div>
