<script lang="ts">
  import { page } from '$app/stores'
  import SkillEditorPage from '$lib/features/skills/components/skill-editor-page.svelte'
  import { pageTitle } from '$lib/i18n'
  import { i18nStore } from '$lib/i18n/store.svelte'

  const skillId = $derived($page.params.skillId ?? '')
</script>

<svelte:head>
  <title>{pageTitle(i18nStore.t('nav.skills'), i18nStore.locale)}</title>
</svelte:head>

<SkillEditorPage {skillId} />
