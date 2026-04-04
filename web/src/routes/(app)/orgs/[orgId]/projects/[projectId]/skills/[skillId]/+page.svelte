<script lang="ts">
  import { page } from '$app/stores'
  import SkillEditorPage from '$lib/features/skills/components/skill-editor-page.svelte'

  const skillId = $derived($page.params.skillId ?? '')
</script>

<svelte:head>
  <title>Skill Editor - OpenASE</title>
</svelte:head>

<SkillEditorPage {skillId} />
