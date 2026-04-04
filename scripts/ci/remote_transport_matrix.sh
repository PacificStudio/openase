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
  "Reverse websocket daemon auth failure" \
  "./internal/httpapi" \
  'TestMachineConnectWebsocketAuthFailurePublishesActivityAndMetric$'

run_case \
  "Listener websocket runtime happy path" \
  "./internal/orchestrator" \
  'TestRuntimeLauncherLaunchesWebsocketListenerRuntimeWithHooksAndArtifactSync$'

run_case \
  "Listener websocket preflight failure classification" \
  "./internal/orchestrator" \
  'TestRuntimeLauncherRecordsWebsocketPreflightFailureStageInActivityAndMetrics$'

run_case \
  "Reverse websocket rollout fallback to SSH" \
  "./internal/orchestrator" \
  'TestRuntimeLauncherFallsBackToSSHWhenWebsocketReverseTransportUnavailable$'

run_case \
  "Pure SSH regression" \
  "./internal/orchestrator" \
  'TestRuntimeLauncherRunTickPreparesRemoteWorkspaceAndLaunchesOverSSH$'
