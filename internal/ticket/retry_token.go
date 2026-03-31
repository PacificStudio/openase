package ticket

import "github.com/google/uuid"

// NewRetryToken returns a new retry-generation token for ticket retry state.
func NewRetryToken() string {
	return uuid.NewString()
}
