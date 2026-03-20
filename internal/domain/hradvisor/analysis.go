package hradvisor

import "time"

type Snapshot struct {
	Project         ProjectContext
	Tickets         []TicketContext
	Workflows       []WorkflowContext
	Agents          []AgentContext
	RecentActivity  []ActivityContext
	ActiveRoleSlugs []string
}

type ProjectContext struct {
	Name                string
	Description         string
	Status              string
	MaxConcurrentAgents int
}

type TicketContext struct {
	Identifier        string
	Type              string
	StatusName        string
	WorkflowType      string
	ConsecutiveErrors int
	RetryPaused       bool
}

type WorkflowContext struct {
	Name     string
	Type     string
	IsActive bool
}

type AgentContext struct {
	Status string
}

type ActivityContext struct {
	EventType string
	Message   string
	CreatedAt time.Time
}

type Analysis struct {
	Summary         Summary
	Recommendations []Recommendation
	Staffing        StaffingPlan
}

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

type Recommendation struct {
	RoleSlug              string
	Priority              string
	Reason                string
	Evidence              []string
	SuggestedHeadcount    int
	SuggestedWorkflowName string
}

type StaffingPlan struct {
	Developers int
	QA         int
	Docs       int
	Security   int
	Product    int
	Research   int
}
