// Package apperr defines typed errors that services return.
// Handlers never decide HTTP status codes themselves; response.Error maps these.
package apperr

import (
	"errors"
	"net/http"
)

type Error struct {
	Status  int               `json:"-"`
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

func Validation(msg string, details map[string]string) *Error {
	return &Error{Status: http.StatusBadRequest, Code: "VALIDATION_ERROR", Message: msg, Details: details}
}

func Unauthenticated(msg string) *Error {
	return &Error{Status: http.StatusUnauthorized, Code: "UNAUTHENTICATED", Message: msg}
}

func Forbidden(msg string) *Error {
	return &Error{Status: http.StatusForbidden, Code: "FORBIDDEN", Message: msg}
}

func NotFound(what string) *Error {
	return &Error{Status: http.StatusNotFound, Code: "NOT_FOUND", Message: what + " not found"}
}

func Conflict(msg string) *Error {
	return &Error{Status: http.StatusConflict, Code: "CONFLICT", Message: msg}
}

// InvalidState: an illegal lifecycle transition, e.g. confirming a closed quotation.
func InvalidState(msg string) *Error {
	return &Error{Status: http.StatusConflict, Code: "INVALID_STATE", Message: msg}
}

// Rule: a business rule was broken, e.g. pausing a plan that is not pausable.
// ruleCode is placed in details.rule so Flutter can react to specific cases.
func Rule(ruleCode, msg string) *Error {
	return &Error{Status: http.StatusUnprocessableEntity, Code: "BUSINESS_RULE", Message: msg,
		Details: map[string]string{"rule": ruleCode}}
}

func NotImplemented(what string) *Error {
	return &Error{Status: http.StatusNotImplemented, Code: "NOT_IMPLEMENTED", Message: what + " is not implemented yet"}
}

// As unwraps err into *Error if possible.
func As(err error) (*Error, bool) {
	var e *Error
	ok := errors.As(err, &e)
	return e, ok
}
