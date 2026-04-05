#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

if [ -x "$ROOT_DIR/.tooling/go/bin/go" ]; then
  export PATH="$ROOT_DIR/.tooling/go/bin:$PATH"
elif [ -x "$HOME/.local/go1.26.1/bin/go" ]; then
  export PATH="$HOME/.local/go1.26.1/bin:$PATH"
fi

run_case() {
  local name="$1"
  local pkg="$2"
  local pattern="$3"

  printf '\n== %s ==\n' "$name"
  (
    cd "$ROOT_DIR"
    go test "$pkg" -run "$pattern"
  )
}

run_case \
  "Reverse websocket machine session observability" \
  "./internal/httpapi" \
  'TestMachineConnectWebsocketPublishesActivityAndMetrics$'

run_case \
  "Unified websocket runtime contract suite" \
  "./internal/infra/machinetransport" \
  'TestUnifiedWebsocketRuntimeContractSuite$'

run_case \
  "Reverse websocket daemon auth failure" \
  "./internal/httpapi" \
  'TestMachineConnectWebsocketAuthFailurePublishesActivityAndMetric$'

run_case \
  "SSH bootstrap helper behavior" \
  "./internal/cli" \
  'TestRunMachineSSHBootstrapUploadsBinaryEnvAndService$'

run_case \
  "SSH diagnostics helper behavior" \
  "./internal/cli" \
  'TestRunMachineSSHDiagnosticsReportsBootstrapAndRegistrationIssues$'

run_case \
  "Listener websocket runtime happy path" \
  "./internal/orchestrator" \
  'TestRuntimeLauncherLaunchesWebsocketListenerRuntimeWithHooksAndArtifactSync$'
  
run_case \
  "Reverse websocket runtime happy path" \
  "./internal/orchestrator" \
  'TestRuntimeLauncherLaunchesWebsocketReverseRuntimeWithHooksAndArtifactSync$'

run_case \
  "Listener websocket preflight failure classification" \
  "./internal/orchestrator" \
  'TestRuntimeLauncherRecordsWebsocketPreflightFailureStageInActivityAndMetrics$'

run_case \
  "Reverse websocket rollout does not fall back to SSH" \
  "./internal/orchestrator" \
  'TestRuntimeLauncherDoesNotFallBackToSSHWhenWebsocketReverseTransportUnavailable$'

run_case \
  "Direct SSH runtime rejection" \
  "./internal/orchestrator" \
  'TestRuntimeLauncherRunTickRejectsSSHRuntimeExecution$'
