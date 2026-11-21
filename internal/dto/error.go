package dto

import "errors"

const (
	CodeEmptyFields     = "empty_fields"
	CodeEmptyName       = "empty_name"
	CodeBadInput        = "bad_input"
	CodeNotFound        = "not_found"
	CodeLastDeck        = "last_deck"
	CodeSaveFailed      = "save_failed"
	CodeUnsupportedFile = "unsupported_file"
	CodeNotUTF8         = "not_utf8"
	CodeNoSheet         = "no_sheet"
	CodeReadFailed      = "read_failed"
	CodeWriteFailed     = "write_failed"
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

type Known struct {
	Err  error
	Code string
}

func FromError(err error, fallback string, known ...Known) error {
	if err == nil {
		return nil
	}
	var e *Error
	if errors.As(err, &e) {
		return e
	}
	code := fallback
	for _, k := range known {
		if errors.Is(err, k.Err) {
			code = k.Code
			break
		}
	}
	return &Error{
		Code:   code,
		Detail: err.Error(),
	}
}
