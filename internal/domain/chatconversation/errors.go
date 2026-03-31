package chatconversation

import "errors"

var (
	ErrNotFound     = errors.New("chat conversation resource not found")
	ErrConflict     = errors.New("chat conversation resource conflict")
	ErrInvalidInput = errors.New("chat conversation invalid input")
)
