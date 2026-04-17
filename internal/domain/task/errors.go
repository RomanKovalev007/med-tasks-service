package task

import "errors"

var (
	ErrNotFound = errors.New("task not found")
	ErrConflict = errors.New("task status conflict")
)
