import { type ChatDiffPayload, type ProjectConversationWorkspaceFilePreview } from '$lib/api/chat'
import type { PersistedWorkspaceFileDraft } from './project-conversation-workspace-file-drafts'
import {
  buildWorkspaceSelection,
  createWorkspacePatchProposal,
  formatWorkspaceDocument,
  formatWorkspaceSelection,
  type WorkspaceSelectionInput,
} from './project-conversation-workspace-editor-helpers'
import {
  createInitialEditorState,
  type WorkspaceFileEditorState,
} from './project-conversation-workspace-browser-state-helpers'
import { chatT } from './i18n'

export function syncWorkspaceEditorStateFromPreview(input: {
  existing: WorkspaceFileEditorState | null
  persisted: PersistedWorkspaceFileDraft | null
  nextPreview: ProjectConversationWorkspaceFilePreview
}): WorkspaceFileEditorState {
  const { existing, persisted, nextPreview } = input
  if (!existing) {
    if (!persisted) {
      return createInitialEditorState(nextPreview)
    }
    const dirty = persisted.draftContent !== nextPreview.content
    return {
      baseSavedContent: persisted.baseSavedContent,
      baseSavedRevision: persisted.baseSavedRevision,
      latestSavedContent: nextPreview.content,
      latestSavedRevision: nextPreview.revision,
      draftContent: persisted.draftContent,
      dirty,
      savePhase: 'idle',
      externalChange: dirty && persisted.baseSavedRevision !== nextPreview.revision,
      errorMessage: '',
      encoding: nextPreview.encoding,
      lineEnding: nextPreview.lineEnding,
      lastSavedAt: '',
      selection: null,
      pendingPatch: null,
    }
  }

  if (existing.dirty) {
    const latestChanged = existing.latestSavedRevision !== nextPreview.revision
    return {
      ...existing,
      latestSavedContent: nextPreview.content,
      latestSavedRevision: nextPreview.revision,
      dirty: existing.draftContent !== nextPreview.content,
      externalChange: latestChanged || existing.baseSavedRevision !== nextPreview.revision,
      encoding: nextPreview.encoding,
      lineEnding: nextPreview.lineEnding,
      selection: buildWorkspaceSelection(existing.draftContent, existing.selection),
    }
  }

  return {
    ...existing,
    baseSavedContent: nextPreview.content,
    baseSavedRevision: nextPreview.revision,
    latestSavedContent: nextPreview.content,
    latestSavedRevision: nextPreview.revision,
    draftContent: nextPreview.content,
    dirty: false,
    externalChange: false,
    savePhase: 'idle',
    errorMessage: '',
    encoding: nextPreview.encoding,
    lineEnding: nextPreview.lineEnding,
    selection: null,
    pendingPatch: null,
  }
}

export function updateWorkspaceEditorDraft(
  editor: WorkspaceFileEditorState,
  nextDraftContent: string,
): WorkspaceFileEditorState {
  return {
    ...editor,
    draftContent: nextDraftContent,
    dirty: nextDraftContent !== editor.latestSavedContent,
    savePhase: editor.savePhase === 'saving' ? editor.savePhase : 'idle',
    errorMessage: '',
    selection: buildWorkspaceSelection(nextDraftContent, editor.selection),
    pendingPatch: null,
  }
}

export function updateWorkspaceEditorSelection(
  editor: WorkspaceFileEditorState,
  selection: WorkspaceSelectionInput | null,
): WorkspaceFileEditorState {
  return {
    ...editor,
    selection: buildWorkspaceSelection(editor.draftContent, selection),
  }
}

export function revertWorkspaceEditorDraft(
  editor: WorkspaceFileEditorState,
): WorkspaceFileEditorState {
  return {
    ...editor,
    baseSavedContent: editor.latestSavedContent,
    baseSavedRevision: editor.latestSavedRevision,
    draftContent: editor.latestSavedContent,
    dirty: false,
    savePhase: 'idle',
    externalChange: false,
    errorMessage: '',
    selection: null,
    pendingPatch: null,
  }
}

export function keepWorkspaceEditorDraft(
  editor: WorkspaceFileEditorState,
): WorkspaceFileEditorState {
  return {
    ...editor,
    baseSavedContent: editor.latestSavedContent,
    baseSavedRevision: editor.latestSavedRevision,
    dirty: editor.draftContent !== editor.latestSavedContent,
    savePhase: 'idle',
    externalChange: false,
    errorMessage: '',
  }
}

export function reviewWorkspaceEditorPatch(input: {
  editor: WorkspaceFileEditorState
  diff: ChatDiffPayload
}): { ok: boolean; nextState: WorkspaceFileEditorState } {
  const proposal = createWorkspacePatchProposal(input.editor.draftContent, input.diff)
  if (!proposal) {
    return {
      ok: false,
      nextState: {
        ...input.editor,
        errorMessage: 'The draft changed and this Project AI patch no longer applies cleanly.',
        pendingPatch: null,
      },
    }
  }

  return {
    ok: true,
    nextState: {
      ...input.editor,
      errorMessage: '',
      pendingPatch: proposal,
    },
  }
}

export function applyWorkspaceEditorPendingPatch(editor: WorkspaceFileEditorState): {
  ok: boolean
  nextState: WorkspaceFileEditorState
} {
  const proposal = editor.pendingPatch
  if (!proposal) {
    return { ok: false, nextState: editor }
  }

  return {
    ok: true,
    nextState: {
      ...editor,
      draftContent: proposal.proposedContent,
      dirty: proposal.proposedContent !== editor.latestSavedContent,
      savePhase: 'idle',
      errorMessage: '',
      selection: buildWorkspaceSelection(proposal.proposedContent, editor.selection),
      pendingPatch: null,
    },
  }
}

export function formatWorkspaceEditorDocument(input: {
  filePath: string
  editor: WorkspaceFileEditorState
}): { ok: boolean; nextState: WorkspaceFileEditorState } {
  try {
    const formatted = formatWorkspaceDocument(input.filePath, input.editor.draftContent)
    if (formatted == null) {
      return {
        ok: false,
        nextState: {
          ...input.editor,
          errorMessage: chatT('chat.formatDocumentUnavailable'),
        },
      }
    }
    return {
      ok: true,
      nextState: {
        ...input.editor,
        draftContent: formatted,
        dirty: formatted !== input.editor.latestSavedContent,
        errorMessage: '',
        selection: buildWorkspaceSelection(formatted, input.editor.selection),
      },
    }
  } catch (error) {
    return {
      ok: false,
      nextState: {
        ...input.editor,
        errorMessage: error instanceof Error ? error.message : chatT('chat.formatDocumentFailed'),
      },
    }
  }
}

export function formatWorkspaceEditorSelection(input: {
  filePath: string
  editor: WorkspaceFileEditorState
}): { ok: boolean; nextState: WorkspaceFileEditorState } {
  try {
    const formatted = formatWorkspaceSelection(
      input.filePath,
      input.editor.draftContent,
      input.editor.selection,
    )
    if (formatted == null) {
      return {
        ok: false,
        nextState: {
          ...input.editor,
          errorMessage: chatT('chat.formatSelectionUnavailable'),
        },
      }
    }
    return {
      ok: true,
      nextState: {
        ...input.editor,
        draftContent: formatted.content,
        dirty: formatted.content !== input.editor.latestSavedContent,
        errorMessage: '',
        selection: buildWorkspaceSelection(formatted.content, formatted.selection),
      },
    }
  } catch (error) {
    return {
      ok: false,
      nextState: {
        ...input.editor,
        errorMessage: error instanceof Error ? error.message : chatT('chat.formatSelectionFailed'),
      },
    }
  }
}
