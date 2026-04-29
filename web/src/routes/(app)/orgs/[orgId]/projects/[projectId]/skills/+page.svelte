<script lang="ts">
  import SkillsPage from '$lib/features/skills/components/skills-page.svelte'
  import { pageTitle } from '$lib/i18n'
  import { i18nStore } from '$lib/i18n/store.svelte'
</script>

<svelte:head>
  <title>{pageTitle(i18nStore.t('nav.skills'), i18nStore.locale)}</title>
</svelte:head>

<SkillsPage />
