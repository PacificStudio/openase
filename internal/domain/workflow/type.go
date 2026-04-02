package workflow

import (
	"fmt"
	"strings"
)

type Type string

const (
	TypeCoding        Type = "coding"
	TypeTest          Type = "test"
	TypeDoc           Type = "doc"
	TypeSecurity      Type = "security"
	TypeDeploy        Type = "deploy"
	TypeRefineHarness Type = "refine-harness"
	TypeCustom        Type = "custom"
)

func ParseType(raw string) (Type, error) {
	workflowType := Type(strings.ToLower(strings.TrimSpace(raw)))
	switch workflowType {
	case TypeCoding, TypeTest, TypeDoc, TypeSecurity, TypeDeploy, TypeRefineHarness, TypeCustom:
		return workflowType, nil
	default:
		return "", fmt.Errorf("type must be one of coding, test, doc, security, deploy, refine-harness, custom")
	}
}

func (t Type) String() string {
	return string(t)
}
