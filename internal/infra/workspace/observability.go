package workspace

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"
)

const remotePreparePhasePrefix = "__OPENASE_REPO_PHASE__|"

type PrepareObservability struct {
	MachineID string
	RunID     string
	TicketID  string
}

func newWorkspaceLogger(base *slog.Logger, component string) *slog.Logger {
	if base == nil {
		base = slog.Default()
	}
	return base.With("component", component)
}

func logRepoPreparePhase(
	logger *slog.Logger,
	observability PrepareObservability,
	repoName string,
	repoPath string,
	phase string,
	duration time.Duration,
	extra ...any,
) {
	if logger == nil {
		return
	}
	attrs := []any{
		"machine_id", strings.TrimSpace(observability.MachineID),
		"run_id", strings.TrimSpace(observability.RunID),
		"ticket_id", strings.TrimSpace(observability.TicketID),
		"repo_name", strings.TrimSpace(repoName),
		"repo_path", strings.TrimSpace(repoPath),
		"phase", strings.TrimSpace(phase),
		"duration_ms", duration.Milliseconds(),
		"duration", duration.String(),
	}
	attrs = append(attrs, extra...)
	logger.Info("workspace repo prepare phase", attrs...)
}

func logRemotePreparePhases(logger *slog.Logger, output []byte) {
	if logger == nil || len(output) == 0 {
		return
	}

	lines := strings.Split(strings.ReplaceAll(string(output), "\r\n", "\n"), "\n")
	for _, line := range lines {
		if !strings.HasPrefix(line, remotePreparePhasePrefix) {
			continue
		}
		fields := strings.Split(line[len(remotePreparePhasePrefix):], "|")
		if len(fields) < 7 {
			logger.Warn("workspace remote prepare phase log malformed", "line", line)
			continue
		}

		durationMS, err := strconv.ParseInt(strings.TrimSpace(fields[6]), 10, 64)
		if err != nil {
			logger.Warn("workspace remote prepare phase duration malformed", "line", line, "error", err)
			continue
		}

		extra := make([]any, 0, 2)
		if len(fields) > 7 && strings.TrimSpace(fields[7]) != "" {
			extra = append(extra, "phase_result", strings.TrimSpace(fields[7]))
		}
		if len(fields) > 8 && strings.TrimSpace(fields[8]) != "" {
			extra = append(extra, "note", strings.TrimSpace(fields[8]))
		}

		logRepoPreparePhase(
			logger,
			PrepareObservability{
				MachineID: fields[0],
				RunID:     fields[1],
				TicketID:  fields[2],
			},
			fields[3],
			fields[4],
			fields[5],
			time.Duration(durationMS)*time.Millisecond,
			extra...,
		)
	}
}

func remotePreparePhaseLine(
	observability PrepareObservability,
	repoName string,
	repoPath string,
	phase string,
	durationMS int64,
	phaseResult string,
	note string,
) string {
	return fmt.Sprintf(
		"%s%s|%s|%s|%s|%s|%s|%d|%s|%s",
		remotePreparePhasePrefix,
		strings.TrimSpace(observability.MachineID),
		strings.TrimSpace(observability.RunID),
		strings.TrimSpace(observability.TicketID),
		strings.TrimSpace(repoName),
		strings.TrimSpace(repoPath),
		strings.TrimSpace(phase),
		durationMS,
		strings.TrimSpace(phaseResult),
		strings.TrimSpace(note),
	)
}
