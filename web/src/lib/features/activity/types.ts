export type ActivityEntry = {
  id: string
  eventType: string
  message: string
  timestamp: string
  ticketIdentifier?: string
  agentName?: string
  metadata?: Record<string, string>
}
