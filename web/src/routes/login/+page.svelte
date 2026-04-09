<script lang="ts">
  import LoginGatePage from '$lib/features/auth/components/login-gate-page.svelte'
  import type { PageData } from './$types'

  let { data }: { data: PageData } = $props()
</script>

<LoginGatePage {data} />
