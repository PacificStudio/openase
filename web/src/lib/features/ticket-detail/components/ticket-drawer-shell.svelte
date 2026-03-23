<script lang="ts">
  import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from '$ui/sheet'

  let {
    open = $bindable(false),
    title = 'Ticket detail',
    loading = false,
    error = '',
  }: {
    open?: boolean
    title?: string
    loading?: boolean
    error?: string
  } = $props()
</script>

<Sheet bind:open>
  <SheetContent side="right" class="flex w-full flex-col p-0 sm:max-w-xl" showCloseButton={false}>
    <SheetHeader class="sr-only">
      <SheetTitle>{title}</SheetTitle>
      <SheetDescription>Ticket detail drawer</SheetDescription>
    </SheetHeader>

    {#if loading}
      <div class="text-muted-foreground flex flex-1 items-center justify-center text-sm">
        Loading ticket detail…
      </div>
    {:else if error}
      <div
        class="text-destructive flex flex-1 items-center justify-center px-6 text-center text-sm"
      >
        {error}
      </div>
    {:else}
      <slot />
    {/if}
  </SheetContent>
</Sheet>
