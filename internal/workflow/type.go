package workflow

import domain "github.com/BetterAndBetterII/openase/internal/domain/workflow"

type TypeLabel = domain.TypeLabel
type Type = domain.Type

const (
	TypeCoding        = domain.TypeCoding
	TypeTest          = domain.TypeTest
	TypeDoc           = domain.TypeDoc
	TypeSecurity      = domain.TypeSecurity
	TypeDeploy        = domain.TypeDeploy
	TypeRefineHarness = domain.TypeRefineHarness
	TypeCustom        = domain.TypeCustom
)

type WorkflowFamily = domain.WorkflowFamily

const (
	WorkflowFamilyPlanning    = domain.WorkflowFamilyPlanning
	WorkflowFamilyDispatcher  = domain.WorkflowFamilyDispatcher
	WorkflowFamilyCoding      = domain.WorkflowFamilyCoding
	WorkflowFamilyReview      = domain.WorkflowFamilyReview
	WorkflowFamilyTest        = domain.WorkflowFamilyTest
	WorkflowFamilyDocs        = domain.WorkflowFamilyDocs
	WorkflowFamilyDeploy      = domain.WorkflowFamilyDeploy
	WorkflowFamilySecurity    = domain.WorkflowFamilySecurity
	WorkflowFamilyHarness     = domain.WorkflowFamilyHarness
	WorkflowFamilyEnvironment = domain.WorkflowFamilyEnvironment
	WorkflowFamilyResearch    = domain.WorkflowFamilyResearch
	WorkflowFamilyReporting   = domain.WorkflowFamilyReporting
	WorkflowFamilyUnknown     = domain.WorkflowFamilyUnknown
)

type WorkflowClassification = domain.WorkflowClassification
type WorkflowClassificationInput = domain.WorkflowClassificationInput

func ParseTypeLabel(raw string) (TypeLabel, error) {
	return domain.ParseTypeLabel(raw)
}

func MustParseTypeLabel(raw string) TypeLabel {
	return domain.MustParseTypeLabel(raw)
}

func ParseType(raw string) (Type, error) {
	return domain.ParseType(raw)
}

func ClassifyWorkflow(input WorkflowClassificationInput) WorkflowClassification {
	return domain.ClassifyWorkflow(input)
}
