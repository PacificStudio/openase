<script lang="ts">
  import { i18nStore } from '$lib/i18n/store.svelte'
  const t = i18nStore.t
</script>

<div>
  <h2 class="text-foreground text-base font-semibold">
    {t('orgAdmin.credentials.intro.heading')}
  </h2>
  <p class="text-muted-foreground mt-1 text-sm">
    {t('orgAdmin.credentials.intro.description')}
  </p>
</div>
