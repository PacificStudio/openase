<script lang="ts">
  import StructuredDiffPreview from '$lib/features/chat/structured-diff-preview.svelte'
  import type { DiffPreview } from '../assistant'

  let {
    preview,
  }: {
    preview: DiffPreview
  } = $props()
</script>

<StructuredDiffPreview {preview} />
