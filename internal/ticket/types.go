package ticket

import domain "github.com/BetterAndBetterII/openase/internal/domain/ticket"

type Priority = domain.Priority

const (
	DefaultPriority = domain.DefaultPriority
	PriorityUrgent  = domain.PriorityUrgent
	PriorityHigh    = domain.PriorityHigh
	PriorityMedium  = domain.PriorityMedium
	PriorityLow     = domain.PriorityLow
)

func ParsePriority(raw string) (Priority, error) {
	return domain.ParsePriority(raw)
}

type Type = domain.Type

const (
	DefaultType  = domain.DefaultType
	TypeFeature  = domain.TypeFeature
	TypeBugfix   = domain.TypeBugfix
	TypeRefactor = domain.TypeRefactor
	TypeChore    = domain.TypeChore
	TypeEpic     = domain.TypeEpic
)

func ParseType(raw string) (Type, error) {
	return domain.ParseType(raw)
}

type DependencyType = domain.DependencyType

const (
	DefaultDependencyType  = domain.DefaultDependencyType
	DependencyTypeBlocks   = domain.DependencyTypeBlocks
	DependencyTypeSubIssue = domain.DependencyTypeSubIssue
)

func ParseDependencyType(raw string) (DependencyType, error) {
	return domain.ParseDependencyType(raw)
}

type ExternalLinkType = domain.ExternalLinkType

const (
	ExternalLinkTypeGithubIssue = domain.ExternalLinkTypeGithubIssue
	ExternalLinkTypeGitlabIssue = domain.ExternalLinkTypeGitlabIssue
	ExternalLinkTypeJiraTicket  = domain.ExternalLinkTypeJiraTicket
	ExternalLinkTypeGithubPR    = domain.ExternalLinkTypeGithubPR
	ExternalLinkTypeGitlabMR    = domain.ExternalLinkTypeGitlabMR
	ExternalLinkTypeCustom      = domain.ExternalLinkTypeCustom
)

func ParseExternalLinkType(raw string) (ExternalLinkType, error) {
	return domain.ParseExternalLinkType(raw)
}

type ExternalLinkRelation = domain.ExternalLinkRelation

const (
	DefaultExternalLinkRelation  = domain.DefaultExternalLinkRelation
	ExternalLinkRelationResolves = domain.ExternalLinkRelationResolves
	ExternalLinkRelationRelated  = domain.ExternalLinkRelationRelated
	ExternalLinkRelationCausedBy = domain.ExternalLinkRelationCausedBy
)

func ParseExternalLinkRelation(raw string) (ExternalLinkRelation, error) {
	return domain.ParseExternalLinkRelation(raw)
}
