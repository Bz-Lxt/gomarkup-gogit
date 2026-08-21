package git

import "errors"

var (
	ErrNotFound        = errors.New("not found")
	ErrValidation      = errors.New("validation error")
	ErrInvalidPath     = errors.New("invalid path")
	ErrAlreadyExists   = errors.New("already exists")
	ErrConflict        = errors.New("conflict")
	ErrMergeConflict   = errors.New("merge conflict")
	ErrAlreadyUpToDate = errors.New("already up to date")
	ErrMergeInProgress = errors.New("merge in progress")
	ErrUnbornHEAD      = errors.New("unborn HEAD")
	ErrDirtyWorktree   = errors.New("uncommitted changes would be overwritten")
)
