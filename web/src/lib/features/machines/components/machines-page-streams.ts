import { connectEventStream } from '$lib/api/sse'

export function connectMachinesPageStream(orgId: string, onEvent: () => void): () => void {
  return connectEventStream(`/api/v1/orgs/${orgId}/machines/stream`, {
    onEvent,
    onError: (streamError) => {
      console.error('Machines stream error:', streamError)
    },
  })
}
