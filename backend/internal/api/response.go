package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"gogit/internal/git"
)

type envelope struct {
	Data  any    `json:"data,omitempty"`
	Error *eBody `json:"error,omitempty"`
}

type eBody struct {
	Code    string   `json:"code"`
	Message string   `json:"message"`
	Details []string `json:"details,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) ok(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusOK, envelope{Data: data})
}

func (s *Server) created(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusCreated, envelope{Data: data})
}

func (s *Server) fail(w http.ResponseWriter, err error) {
	status, code, details := mapErr(err)
	s.log.Warn("api error", "code", code, "err", err.Error())
	writeJSON(w, status, envelope{Error: &eBody{Code: code, Message: err.Error(), Details: details}})
}

func mapErr(err error) (int, string, []string) {
	switch {
	case errors.Is(err, git.ErrNotFound), errors.Is(err, git.ErrUnbornHEAD):
		return http.StatusNotFound, "not_found", nil
	case errors.Is(err, git.ErrInvalidPath):
		return http.StatusBadRequest, "invalid_path", nil
	case errors.Is(err, git.ErrValidation):
		return http.StatusBadRequest, "validation_error", nil
	case errors.Is(err, git.ErrAlreadyExists):
		return http.StatusConflict, "already_exists", nil
	case errors.Is(err, git.ErrAlreadyUpToDate):
		return http.StatusConflict, "already_up_to_date", nil
	case errors.Is(err, git.ErrMergeInProgress):
		return http.StatusUnprocessableEntity, "merge_in_progress", nil
	case errors.Is(err, git.ErrDirtyWorktree):
		return http.StatusConflict, "conflict", nil
	case errors.Is(err, git.ErrMergeConflict):
		return http.StatusConflict, "conflict", nil
	case errors.Is(err, git.ErrConflict):
		return http.StatusConflict, "conflict", nil
	default:
		return http.StatusInternalServerError, "internal_error", nil
	}
}

func decodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("%w: invalid json", git.ErrValidation)
	}
	return nil
}
