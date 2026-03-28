export type ActionProposalPathMatch =
  | { kind: 'projectTickets'; projectId: string }
  | { kind: 'projectWorkflows'; projectId: string }
  | { kind: 'project'; projectId: string }
  | { kind: 'ticket'; ticketId: string }
  | { kind: 'ticketComments'; ticketId: string }
  | { kind: 'ticketComment'; ticketId: string; commentId: string }
  | { kind: 'ticketDependencies'; ticketId: string }
  | { kind: 'ticketDependency'; ticketId: string; dependencyId: string }
  | { kind: 'ticketExternalLinks'; ticketId: string }
  | { kind: 'ticketExternalLink'; ticketId: string; externalLinkId: string }
  | { kind: 'workflow'; workflowId: string }
  | { kind: 'workflowHarness'; workflowId: string }
  | { kind: 'workflowBindSkills'; workflowId: string }
  | { kind: 'workflowUnbindSkills'; workflowId: string }
  | { kind: 'unknown' }

export function matchActionProposalPath(path: string): ActionProposalPathMatch {
  const projectTickets = path.match(/^\/api\/v1\/projects\/([^/]+)\/tickets$/)
  if (projectTickets) {
    return { kind: 'projectTickets', projectId: decodeURIComponent(projectTickets[1]) }
  }

  const projectWorkflows = path.match(/^\/api\/v1\/projects\/([^/]+)\/workflows$/)
  if (projectWorkflows) {
    return { kind: 'projectWorkflows', projectId: decodeURIComponent(projectWorkflows[1]) }
  }

  const project = path.match(/^\/api\/v1\/projects\/([^/]+)$/)
  if (project) {
    return { kind: 'project', projectId: decodeURIComponent(project[1]) }
  }

  const ticketComment = path.match(/^\/api\/v1\/tickets\/([^/]+)\/comments\/([^/]+)$/)
  if (ticketComment) {
    return {
      kind: 'ticketComment',
      ticketId: decodeURIComponent(ticketComment[1]),
      commentId: decodeURIComponent(ticketComment[2]),
    }
  }

  const ticketComments = path.match(/^\/api\/v1\/tickets\/([^/]+)\/comments$/)
  if (ticketComments) {
    return { kind: 'ticketComments', ticketId: decodeURIComponent(ticketComments[1]) }
  }

  const ticketDependency = path.match(/^\/api\/v1\/tickets\/([^/]+)\/dependencies\/([^/]+)$/)
  if (ticketDependency) {
    return {
      kind: 'ticketDependency',
      ticketId: decodeURIComponent(ticketDependency[1]),
      dependencyId: decodeURIComponent(ticketDependency[2]),
    }
  }

  const ticketDependencies = path.match(/^\/api\/v1\/tickets\/([^/]+)\/dependencies$/)
  if (ticketDependencies) {
    return { kind: 'ticketDependencies', ticketId: decodeURIComponent(ticketDependencies[1]) }
  }

  const ticketExternalLink = path.match(/^\/api\/v1\/tickets\/([^/]+)\/external-links\/([^/]+)$/)
  if (ticketExternalLink) {
    return {
      kind: 'ticketExternalLink',
      ticketId: decodeURIComponent(ticketExternalLink[1]),
      externalLinkId: decodeURIComponent(ticketExternalLink[2]),
    }
  }

  const ticketExternalLinks = path.match(/^\/api\/v1\/tickets\/([^/]+)\/external-links$/)
  if (ticketExternalLinks) {
    return { kind: 'ticketExternalLinks', ticketId: decodeURIComponent(ticketExternalLinks[1]) }
  }

  const ticket = path.match(/^\/api\/v1\/tickets\/([^/]+)$/)
  if (ticket) {
    return { kind: 'ticket', ticketId: decodeURIComponent(ticket[1]) }
  }

  const workflowHarness = path.match(/^\/api\/v1\/workflows\/([^/]+)\/harness$/)
  if (workflowHarness) {
    return { kind: 'workflowHarness', workflowId: decodeURIComponent(workflowHarness[1]) }
  }

  const workflowBindSkills = path.match(/^\/api\/v1\/workflows\/([^/]+)\/skills\/bind$/)
  if (workflowBindSkills) {
    return {
      kind: 'workflowBindSkills',
      workflowId: decodeURIComponent(workflowBindSkills[1]),
    }
  }

  const workflowUnbindSkills = path.match(/^\/api\/v1\/workflows\/([^/]+)\/skills\/unbind$/)
  if (workflowUnbindSkills) {
    return {
      kind: 'workflowUnbindSkills',
      workflowId: decodeURIComponent(workflowUnbindSkills[1]),
    }
  }

  const workflow = path.match(/^\/api\/v1\/workflows\/([^/]+)$/)
  if (workflow) {
    return { kind: 'workflow', workflowId: decodeURIComponent(workflow[1]) }
  }

  return { kind: 'unknown' }
}
