package dto

import (
	"errors"
	"testing"
)

func TestErrorMessage(t *testing.T) {
	tests := []struct {
		err  *Error
		want string
	}{
		{&Error{Code: CodeNotFound}, "not_found"},
		{&Error{Code: CodeNotFound, Detail: "no card with id 9"}, "not_found: no card with id 9"},
	}
	for _, tt := range tests {
		if got := tt.err.Error(); got != tt.want {
			t.Errorf("Error() = %q, want %q", got, tt.want)
		}
	}
}

func TestIsCode(t *testing.T) {
	if !IsCode(&Error{Code: CodeNotFound}, CodeNotFound) {
		t.Error("IsCode should match an *Error with the same code")
	}
	if IsCode(nil, CodeNotFound) || IsCode(errors.New(CodeNotFound), CodeNotFound) {
		t.Error("IsCode should only match *Error values")
	}
	if IsCode(&Error{Code: CodeBadInput}, CodeNotFound) {
		t.Error("IsCode should compare codes")
	}
}
