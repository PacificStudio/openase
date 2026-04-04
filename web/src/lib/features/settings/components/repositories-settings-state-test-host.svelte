<script lang="ts">
  import { createRepositoriesSettingsState } from './repositories-settings-state.svelte'

  let {
    onReady,
  }: { onReady?: (state: ReturnType<typeof createRepositoriesSettingsState>) => void } = $props()

  const state = createRepositoriesSettingsState()

  $effect(() => {
    onReady?.(state)
  })
</script>
