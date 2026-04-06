import { describe, expect, it } from 'vitest'

import { buildEventCatalog, getSeverity, severityLabel } from './notification-event-catalog'

const sampleEventTypes = [
  { event_type: 'ticket.created', label: 'Ticket Created', group: 'Ticket lifecycle', level: 'info', default_template: '' },
  { event_type: 'ticket.status_changed', label: 'Ticket Status Changed', group: 'Ticket lifecycle', level: 'info', default_template: '' },
  { event_type: 'ticket.budget_exhausted', label: 'Ticket Budget Exhausted', group: 'Ticket health', level: 'critical', default_template: '' },
  { event_type: 'agent.failed', label: 'Agent Failed', group: 'Agents', level: 'critical', default_template: '' },
  { event_type: 'machine.offline', label: 'Machine Offline', group: 'Infrastructure', level: 'critical', default_template: '' },
  { event_type: 'pr.opened', label: 'PR Opened', group: 'Pull requests', level: 'info', default_template: '' },
]

describe('notification event catalog', () => {
  it('groups events by backend-provided group field', () => {
    const groups = buildEventCatalog(sampleEventTypes)
    const labels = groups.map((g) => g.label)

    expect(labels).toContain('Ticket lifecycle')
    expect(labels).toContain('Ticket health')
    expect(labels).toContain('Agents')
    expect(labels).toContain('Pull requests')
    expect(labels).toContain('Infrastructure')
  })

  it('assigns severity from backend level field', () => {
    expect(getSeverity('ticket.created', sampleEventTypes)).toBe('info')
    expect(getSeverity('ticket.budget_exhausted', sampleEventTypes)).toBe('critical')
    expect(getSeverity('agent.failed', sampleEventTypes)).toBe('critical')
    expect(getSeverity('unknown.event', sampleEventTypes)).toBe('info')
  })

  it('returns human-readable severity labels', () => {
    expect(severityLabel('info')).toBe('Info')
    expect(severityLabel('warning')).toBe('Warning')
    expect(severityLabel('critical')).toBe('Critical')
  })

  it('places events in their declared group', () => {
    const groups = buildEventCatalog(sampleEventTypes)
    const lifecycle = groups.find((g) => g.label === 'Ticket lifecycle')
    expect(lifecycle).toBeDefined()
    expect(lifecycle!.events.some((e) => e.eventType === 'ticket.created')).toBe(true)
    expect(lifecycle!.events.some((e) => e.eventType === 'ticket.status_changed')).toBe(true)
  })

  it('omits groups with no matching events', () => {
    const groups = buildEventCatalog([
      { event_type: 'ticket.created', label: 'Ticket Created', group: 'Ticket lifecycle', level: 'info', default_template: '' },
    ])
    const labels = groups.map((g) => g.label)

    expect(labels).toContain('Ticket lifecycle')
    expect(labels).not.toContain('Infrastructure')
    expect(labels).not.toContain('Agents')
  })

  it('handles events without a group in an Other group', () => {
    const groups = buildEventCatalog([
      { event_type: 'custom.event', label: 'Custom Event', group: '', level: '', default_template: '' },
    ])

    const other = groups.find((g) => g.label === 'Other')
    expect(other).toBeDefined()
    expect(other!.events[0].eventType).toBe('custom.event')
  })
})
