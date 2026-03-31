package catalog

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type WorkspaceDashboardSummary struct {
	OrganizationCount int
	ProjectCount      int
	ProviderCount     int
	RunningAgents     int
	ActiveTickets     int
	TodayCost         float64
	TotalTokens       int64
	Organizations     []WorkspaceOrganizationSummary
}

type WorkspaceOrganizationSummary struct {
	OrganizationID uuid.UUID
	Name           string
	Slug           string
	ProjectCount   int
	ProviderCount  int
	RunningAgents  int
	ActiveTickets  int
	TodayCost      float64
	TotalTokens    int64
}

type OrganizationDashboardSummary struct {
	OrganizationID     uuid.UUID
	ProjectCount       int
	ActiveProjectCount int
	ProviderCount      int
	RunningAgents      int
	ActiveTickets      int
	TodayCost          float64
	TotalTokens        int64
	Projects           []OrganizationProjectSummary
}

type OrganizationProjectSummary struct {
	ProjectID      uuid.UUID
	Name           string
	Description    string
	Status         string
	RunningAgents  int
	ActiveTickets  int
	TodayCost      float64
	TotalTokens    int64
	LastActivityAt *time.Time
}

func IsTerminalTicketStatusName(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "done", "cancelled", "canceled", "archived":
		return true
	default:
		return false
	}
}

func IsActiveProjectStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "archived", "cancelled", "canceled":
		return false
	default:
		return true
	}
}
