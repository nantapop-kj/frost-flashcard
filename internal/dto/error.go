package dto

import "errors"

const (
	CodeEmptyFields = "empty_fields"
	CodeBadInput    = "bad_input"
	CodeNotFound    = "not_found"
	CodeSaveFailed  = "save_failed"
)

type Error struct {
	Code   string `json:"code"`
	Detail string `json:"detail,omitempty"`
}

func (e *Error) Error() string {
	if e.Detail == "" {
		return e.Code
	}
	return e.Code + ": " + e.Detail
}

func IsCode(err error, code string) bool {
	var e *Error
	return errors.As(err, &e) && e.Code == code
}
