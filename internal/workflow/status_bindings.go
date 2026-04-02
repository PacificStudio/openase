package workflow

import (
	domain "github.com/BetterAndBetterII/openase/internal/domain/workflow"
	"github.com/google/uuid"
)

type StatusBindingSet = domain.StatusBindingSet

func ParseStatusBindingSet(fieldName string, raw []uuid.UUID) (StatusBindingSet, error) {
	return domain.ParseStatusBindingSet(fieldName, raw)
}

func MustStatusBindingSet(ids ...uuid.UUID) StatusBindingSet {
	return domain.MustStatusBindingSet(ids...)
}
