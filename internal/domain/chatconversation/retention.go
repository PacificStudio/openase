package chatconversation

import (
	"time"

	"github.com/google/uuid"
)

type DeleteTrigger string

const (
	DeleteTriggerManual    DeleteTrigger = "manual"
	DeleteTriggerRetention DeleteTrigger = "retention"
)

type DeleteConversationInput struct {
	Force   bool
	Trigger DeleteTrigger
}

type DeleteConversationResult struct {
	ConversationID     uuid.UUID
	ProjectID          uuid.UUID
	UserID             string
	Trigger            DeleteTrigger
	WorkspacePath      string
	WorkspaceDeleted   bool
	WorkspaceDirty     bool
	EntriesDeleted     int
	TurnsDeleted       int
	InterruptsDeleted  int
	RunsDeleted        int
	TraceEventsDeleted int
	StepEventsDeleted  int
	AgentTokensDeleted int
	DeletedAt          time.Time
}

type RetentionCleanupSkipReason string

const (
	RetentionCleanupSkipLatestWindow   RetentionCleanupSkipReason = "latest_window"
	RetentionCleanupSkipRecentActivity RetentionCleanupSkipReason = "recent_activity"
	RetentionCleanupSkipRuntimeActive  RetentionCleanupSkipReason = "runtime_active"
	RetentionCleanupSkipPendingInput   RetentionCleanupSkipReason = "pending_interrupt"
	RetentionCleanupSkipDirtyWorkspace RetentionCleanupSkipReason = "dirty_workspace"
)

type RetentionCleanupSkip struct {
	ConversationID uuid.UUID
	ProjectID      uuid.UUID
	UserID         string
	Reason         RetentionCleanupSkipReason
	Detail         string
}

type RetentionCleanupResult struct {
	ProjectID uuid.UUID
	Scanned   int
	Deleted   []DeleteConversationResult
	Skipped   []RetentionCleanupSkip
	RanAt     time.Time
}
