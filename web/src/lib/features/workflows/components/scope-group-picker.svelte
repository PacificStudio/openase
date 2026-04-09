<script lang="ts">
  import { Checkbox } from '$ui/checkbox'
  import { ChevronRight } from '@lucide/svelte'
  import type { ScopeGroup } from '../types'

  let {
    groups = [],
    selected = [],
    lockedScopes = [],
    disabled = false,
    onchange,
  }: {
    groups?: ScopeGroup[]
    selected?: string[]
    lockedScopes?: string[]
    disabled?: boolean
    onchange?: (scopes: string[]) => void
  } = $props()

  let expandedCategories = $state<Set<string>>(new Set())

  const selectedSet = $derived(new Set(selected))
  const lockedScopeSet = $derived(new Set(lockedScopes))

  function toggleCategory(category: string) {
    const next = new Set(expandedCategories)
    if (next.has(category)) {
      next.delete(category)
    } else {
      next.add(category)
    }
    expandedCategories = next
  }

  function toggleScope(scope: string) {
    if (lockedScopeSet.has(scope)) return
    const next = selectedSet.has(scope) ? selected.filter((s) => s !== scope) : [...selected, scope]
    onchange?.(next)
  }

  function toggleGroup(group: ScopeGroup) {
    const mutableScopes = group.scopes.filter((scope) => !lockedScopeSet.has(scope))
    const allSelected =
      mutableScopes.length > 0 && mutableScopes.every((scope) => selectedSet.has(scope))
    let next: string[]
    if (allSelected) {
      const groupSet = new Set(mutableScopes)
      next = selected.filter((s) => !groupSet.has(s))
    } else {
      const merged = new Set(selected)
      for (const scope of group.scopes) merged.add(scope)
      next = [...merged]
    }
    onchange?.(next)
  }

  function groupState(group: ScopeGroup): { checked: boolean; indeterminate: boolean } {
    const count = group.scopes.filter((scope) => selectedSet.has(scope)).length
    if (count === 0) return { checked: false, indeterminate: false }
    if (count === group.scopes.length) return { checked: true, indeterminate: false }
    return { checked: false, indeterminate: true }
  }

  function scopeLabel(scope: string, category: string): string {
    return scope.startsWith(category + '.') ? scope.slice(category.length + 1) : scope
  }
</script>

<div class="divide-border divide-y rounded-md border">
  {#each groups as group (group.category)}
    {@const state = groupState(group)}
    {@const isExpanded = expandedCategories.has(group.category)}
    <div>
      <button
        type="button"
        class="hover:bg-muted/50 flex w-full items-center gap-2 px-3 py-2 text-left transition-colors"
        {disabled}
        onclick={() => toggleCategory(group.category)}
      >
        <ChevronRight
          class="text-muted-foreground size-3.5 shrink-0 transition-transform {isExpanded
            ? 'rotate-90'
            : ''}"
        />
        <!-- svelte-ignore a11y_click_events_have_key_events -->
        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <span
          class="flex items-center"
          onclick={(e) => {
            e.stopPropagation()
            toggleGroup(group)
          }}
        >
          <Checkbox
            checked={state.checked}
            indeterminate={state.indeterminate}
            {disabled}
            onCheckedChange={() => toggleGroup(group)}
          />
        </span>
        <span class="text-sm font-medium capitalize">{group.category.replace(/_/g, ' ')}</span>
        <span class="text-muted-foreground ml-auto text-xs">
          {group.scopes.filter((s) => selectedSet.has(s)).length}/{group.scopes.length}
        </span>
      </button>
      {#if isExpanded}
        <div class="bg-muted/20 border-border space-y-0 border-t px-3 py-1">
          {#each group.scopes as scope (scope)}
            {@const isLocked = lockedScopeSet.has(scope)}
            <label
              class="hover:bg-muted/40 flex cursor-pointer items-center gap-2 rounded px-2 py-1.5 transition-colors"
              class:pointer-events-none={disabled || isLocked}
            >
              <Checkbox
                checked={selectedSet.has(scope)}
                disabled={disabled || isLocked}
                onCheckedChange={() => toggleScope(scope)}
              />
              <span class="text-muted-foreground font-mono text-xs"
                >{scopeLabel(scope, group.category)}</span
              >
              {#if isLocked}
                <span
                  class="text-primary rounded-full border border-current px-1.5 py-0.5 text-[10px] font-medium uppercase"
                >
                  Required
                </span>
              {/if}
            </label>
          {/each}
        </div>
      {/if}
    </div>
  {/each}
</div>
