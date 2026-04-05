import type { TicketRun } from './types'

export function toRunRecord(run: TicketRun) {
  return {
    id: run.id,
    ticket_id: 'ticket-1',
    attempt_number: run.attemptNumber,
    agent_id: run.agentId,
    agent_name: run.agentName,
    provider: run.provider,
    adapter_type: run.adapterType,
    model_name: run.modelName,
    usage: {
      total: run.usage.total,
      input: run.usage.input,
      output: run.usage.output,
      cached_input: run.usage.cachedInput,
      cache_creation: run.usage.cacheCreation,
      reasoning: run.usage.reasoning,
      prompt: run.usage.prompt,
      candidate: run.usage.candidate,
      tool: run.usage.tool,
    },
    status: run.status,
    current_step_status: run.currentStepStatus ?? null,
    current_step_summary: run.currentStepSummary ?? null,
    created_at: run.createdAt,
    runtime_started_at: run.runtimeStartedAt ?? null,
    last_heartbeat_at: run.lastHeartbeatAt ?? null,
    completed_at: run.completedAt ?? null,
    terminal_at: run.terminalAt ?? run.completedAt ?? null,
    last_error: run.lastError ?? null,
    completion_summary: run.completionSummary
      ? {
          status: run.completionSummary.status,
          markdown: run.completionSummary.markdown ?? null,
          json: run.completionSummary.json ?? null,
          generated_at: run.completionSummary.generatedAt ?? null,
          error: run.completionSummary.error ?? null,
        }
      : null,
  }
}
