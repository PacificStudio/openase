package catalog

import "errors"

var (
	ErrNotFound     = errors.New("catalog resource not found")
	ErrConflict     = errors.New("catalog resource conflict")
	ErrInvalidInput = errors.New("catalog invalid input")
)
