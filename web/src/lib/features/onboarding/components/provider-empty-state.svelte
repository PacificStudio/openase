<script lang="ts">
  import { Button } from '$ui/button'
  import { CircleDashed } from '@lucide/svelte'

  let { orgId }: { orgId: string } = $props()
</script>

<div class="border-border rounded-xl border border-dashed p-6 text-center">
  <CircleDashed class="text-muted-foreground mx-auto mb-2 size-8" />
  <p class="text-foreground text-sm font-medium">还没有注册任何 Provider。</p>
  <p class="text-muted-foreground mt-1 text-xs">
    先参考上面的 CLI 指南完成安装与登录，然后在组织设置里注册对应 Provider，回来后点“重新检测全部
    Provider”即可。
  </p>
  <Button
    variant="outline"
    size="sm"
    class="mt-3"
    onclick={() => window.open(`/orgs/${orgId}/settings`, '_blank')}
  >
    前往组织设置
  </Button>
</div>
