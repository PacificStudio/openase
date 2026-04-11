<script lang="ts">
  let { class: className = '' }: { class?: string } = $props()
</script>

<div class={className} data-testid="mock-line-chart"></div>
