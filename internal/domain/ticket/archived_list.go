package ticket

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

const (
	DefaultArchivedTicketPage    = 1
	DefaultArchivedTicketPerPage = 20
	MaxArchivedTicketPerPage     = 100
)

type ArchivedListInput struct {
	ProjectID uuid.UUID
	Page      int
	PerPage   int
}

type ArchivedListResult struct {
	Tickets []Ticket
	Total   int
	Page    int
	PerPage int
}

type ArchivedListRawInput struct {
	Page    string
	PerPage string
}

func ParseArchivedListInput(projectID uuid.UUID, raw ArchivedListRawInput) (ArchivedListInput, error) {
	page, err := parseArchivedListPage(raw.Page)
	if err != nil {
		return ArchivedListInput{}, err
	}
	perPage, err := parseArchivedListPerPage(raw.PerPage)
	if err != nil {
		return ArchivedListInput{}, err
	}

	return ArchivedListInput{
		ProjectID: projectID,
		Page:      page,
		PerPage:   perPage,
	}, nil
}

func parseArchivedListPage(raw string) (int, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return DefaultArchivedTicketPage, nil
	}

	parsed, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0, fmt.Errorf("page must be a valid integer")
	}
	if parsed <= 0 {
		return 0, fmt.Errorf("page must be greater than zero")
	}

	return parsed, nil
}

func parseArchivedListPerPage(raw string) (int, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return DefaultArchivedTicketPerPage, nil
	}

	parsed, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0, fmt.Errorf("per_page must be a valid integer")
	}
	if parsed <= 0 {
		return 0, fmt.Errorf("per_page must be greater than zero")
	}
	if parsed > MaxArchivedTicketPerPage {
		return 0, fmt.Errorf("per_page must be less than or equal to %d", MaxArchivedTicketPerPage)
	}

	return parsed, nil
}
