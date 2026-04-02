package workflow

import domain "github.com/BetterAndBetterII/openase/internal/domain/workflow"

// Type identifies a workflow role/runtime shape without exposing ent enums.
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

// ParseType validates a raw workflow type string.
func ParseType(raw string) (Type, error) {
	return domain.ParseType(raw)
}
