package dto

import (
	"errors"
	"fmt"
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

func TestFromError(t *testing.T) {
	errA := errors.New("a broke")
	known := []Known{{errA, CodeLastDeck}}

	if FromError(nil, CodeSaveFailed, known...) != nil {
		t.Error("nil should stay nil")
	}

	orig := &Error{Code: CodeNotFound}
	if got := FromError(orig, CodeSaveFailed, known...); got != orig {
		t.Errorf("*Error should pass through, got %v", got)
	}

	wrapped := fmt.Errorf("deck 3: %w", errA)
	got := FromError(wrapped, CodeSaveFailed, known...)
	if !IsCode(got, CodeLastDeck) || got.(*Error).Detail != "deck 3: a broke" {
		t.Errorf("known error = %#v", got)
	}

	if got := FromError(errors.New("disk"), CodeSaveFailed, known...); !IsCode(got, CodeSaveFailed) {
		t.Errorf("unknown error = %#v, want fallback", got)
	}
}
