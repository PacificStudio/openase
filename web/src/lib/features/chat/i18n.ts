import type { TranslationKey, TranslationParams } from '$lib/i18n/index'
import { i18nStore } from '$lib/i18n/store.svelte'

export function chatT(key: TranslationKey, params?: TranslationParams) {
  return i18nStore.t(key, params)
}

export function chatWorkspaceStatusT(status: string): string {
  switch (status) {
    case 'added':
      return chatT('chat.workspace.status.added')
    case 'deleted':
      return chatT('chat.workspace.status.deleted')
    case 'modified':
      return chatT('chat.workspace.status.modified')
    case 'renamed':
      return chatT('chat.workspace.status.renamed')
    case 'untracked':
      return chatT('chat.workspace.status.untracked')
    default:
      return status
  }
}
