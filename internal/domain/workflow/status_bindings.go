package workflow

import (
	"fmt"
	"slices"

	"github.com/google/uuid"
)

type StatusBindingSet struct {
	ids []uuid.UUID
}

func ParseStatusBindingSet(fieldName string, raw []uuid.UUID) (StatusBindingSet, error) {
	parsed := make([]uuid.UUID, 0, len(raw))
	for _, id := range raw {
		if id == uuid.Nil {
			return StatusBindingSet{}, fmt.Errorf("%s must not contain zero UUID values", fieldName)
		}
		if !slices.Contains(parsed, id) {
			parsed = append(parsed, id)
		}
	}
	if len(parsed) == 0 {
		return StatusBindingSet{}, fmt.Errorf("%s must contain at least one status", fieldName)
	}

	return StatusBindingSet{ids: parsed}, nil
}

func MustStatusBindingSet(ids ...uuid.UUID) StatusBindingSet {
	parsed, err := ParseStatusBindingSet("status_bindings", ids)
	if err != nil {
		panic(err)
	}
	return parsed
}

func (s StatusBindingSet) IDs() []uuid.UUID {
	return append([]uuid.UUID(nil), s.ids...)
}

func (s StatusBindingSet) Len() int {
	return len(s.ids)
}

func (s StatusBindingSet) Contains(id uuid.UUID) bool {
	return slices.Contains(s.ids, id)
}

func (s StatusBindingSet) Overlaps(other StatusBindingSet) bool {
	for _, id := range s.ids {
		if other.Contains(id) {
			return true
		}
	}
	return false
}

func (s StatusBindingSet) Single() (uuid.UUID, bool) {
	if len(s.ids) != 1 {
		return uuid.UUID{}, false
	}
	return s.ids[0], true
}
