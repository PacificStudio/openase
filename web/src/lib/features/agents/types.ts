export type AgentProvider = {
  id: string
  organization_id: string
  name: string
  adapter_type: string
  cli_command: string
  cli_args: string[]
  auth_config: Record<string, unknown>
  model_name: string
  model_temperature: number
  model_max_tokens: number
  cost_per_input_token: number
  cost_per_output_token: number
}
