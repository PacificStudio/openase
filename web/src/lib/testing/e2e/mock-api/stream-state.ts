export const projectConversationStreamControllers = new Map<
  string,
  Set<ReadableStreamDefaultController<Uint8Array>>
>()
export const queuedProjectConversationFrames = new Map<string, string[]>()
export const projectConversationMuxStreamControllers = new Map<
  string,
  Set<ReadableStreamDefaultController<Uint8Array>>
>()
export const queuedProjectConversationMuxFrames = new Map<string, string[]>()

export function resetMockStreamState() {
  projectConversationStreamControllers.clear()
  queuedProjectConversationFrames.clear()
  projectConversationMuxStreamControllers.clear()
  queuedProjectConversationMuxFrames.clear()
}
