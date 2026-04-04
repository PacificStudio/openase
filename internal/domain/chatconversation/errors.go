package chatconversation

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound          = errors.New("chat conversation resource not found")
	ErrConflict          = errors.New("chat conversation resource conflict")
	ErrInvalidInput      = errors.New("chat conversation invalid input")
	ErrTurnAlreadyActive = fmt.Errorf(
		"%w: project conversation already has an active turn",
		ErrConflict,
	)
	ErrInterruptPending = fmt.Errorf(
		"%w: project conversation has a pending interrupt",
		ErrConflict,
	)
)
