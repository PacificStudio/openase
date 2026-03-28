package hradvisor

import "time"

// Snapshot is the project state analyzed by the HR advisor.
type Snapshot struct {
	Project             ProjectContext
	Tickets             []TicketContext
	Workflows           []WorkflowContext
	Agents              []AgentContext
	RecentActivityCount int
	RecentTrends        []ActivityTrendContext
	ActiveRoleSlugs     []string
}

// ProjectContext summarizes the project-level planning signals.
type ProjectContext struct {
	Name                string
	Description         string
	Status              string
	MaxConcurrentAgents int
}

// TicketContext summarizes one ticket for staffing analysis.
type TicketContext struct {
	Identifier        string
	Type              string
	StatusName        string
	WorkflowType      string
	HasActiveRun      bool
	ConsecutiveErrors int
	RetryPaused       bool
}

// WorkflowContext summarizes one workflow for staffing analysis.
type WorkflowContext struct {
	Name              string
	Type              string
	RoleSlug          string
	IsActive          bool
	PickupStatusNames []string
	FinishStatusNames []string
}

// AgentContext summarizes one agent for staffing analysis.
type AgentContext struct {
	Status string
}

const (
	ActivityTrendDocumentationDrift = "documentation_drift"
	ActivityTrendFailureBurst       = "failure_burst"
)

// ActivityTrendContext summarizes one parsed recent-activity trend.
type ActivityTrendContext struct {
	Kind     string
	Count    int
	Evidence []string
	LastSeen time.Time
}

// Analysis is the staffing recommendation result produced by the advisor.
type Analysis struct {
	Summary         Summary
	Recommendations []Recommendation
	Staffing        StaffingPlan
}

// Summary captures aggregate staffing signals from a snapshot.
type Summary struct {
	OpenTickets         int
	CodingTickets       int
	FailingTickets      int
	BlockedTickets      int
	ActiveAgents        int
	WorkflowCount       int
	RecentActivityCount int
	ActiveWorkflowTypes []string
}

// Recommendation suggests adding or adjusting one role workflow.
type Recommendation struct {
	RoleSlug              string
	Priority              string
	Reason                string
	Evidence              []string
	SuggestedHeadcount    int
	SuggestedWorkflowName string
}

// StaffingPlan estimates headcount split by role family.
type StaffingPlan struct {
	Developers int
	QA         int
	Docs       int
	Security   int
	Product    int
	Research   int
}
