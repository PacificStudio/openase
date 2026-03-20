package pgarray

import (
	"database/sql/driver"

	"github.com/lib/pq"
)

// StringArray stores PostgreSQL text[] values without JSON re-encoding.
type StringArray []string

func (a StringArray) Value() (driver.Value, error) {
	if a == nil {
		return nil, nil
	}

	return pq.StringArray(a).Value()
}

func (a *StringArray) Scan(src any) error {
	if src == nil {
		*a = nil
		return nil
	}

	var raw pq.StringArray
	if err := raw.Scan(src); err != nil {
		return err
	}

	*a = StringArray(raw)
	return nil
}
