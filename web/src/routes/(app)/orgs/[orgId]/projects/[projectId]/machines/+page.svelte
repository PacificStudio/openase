<script lang="ts">
  import { MachinesPage } from '$lib/features/machines'
  import type { PageData } from './$types'

  let { data }: { data: PageData } = $props()
</script>

<svelte:head>
  <title>Machines - OpenASE</title>
</svelte:head>

<MachinesPage {data} />
