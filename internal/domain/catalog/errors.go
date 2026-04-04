package catalog

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound     = errors.New("catalog resource not found")
	ErrConflict     = errors.New("catalog resource conflict")
	ErrInvalidInput = errors.New("catalog invalid input")

	ErrOrganizationSlugConflict = fmt.Errorf("%w: organization slug already exists", ErrConflict)
	ErrProjectSlugConflict      = fmt.Errorf("%w: project slug already exists", ErrConflict)
	ErrMachineNameConflict      = fmt.Errorf("%w: machine name already exists", ErrConflict)
	ErrMachineInUseConflict     = fmt.Errorf("%w: machine is still referenced by other resources", ErrConflict)

	ErrAgentProviderNameConflict    = fmt.Errorf("%w: agent provider name already exists", ErrConflict)
	ErrProjectRepoNameConflict      = fmt.Errorf("%w: repository name already exists", ErrConflict)
	ErrProjectRepoInUseConflict     = fmt.Errorf("%w: repository is still referenced by other resources", ErrConflict)
	ErrTicketRepoScopeConflict      = fmt.Errorf("%w: repository is already attached to this ticket", ErrConflict)
	ErrTicketRepoScopeInUseConflict = fmt.Errorf("%w: ticket repository scope is still referenced by active runtime resources", ErrConflict)

	ErrAgentNameConflict  = fmt.Errorf("%w: agent name already exists", ErrConflict)
	ErrAgentInUseConflict = fmt.Errorf("%w: agent is still referenced by other resources", ErrConflict)
)
