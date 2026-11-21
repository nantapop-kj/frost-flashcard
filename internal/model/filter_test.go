package model

import "testing"

func TestFilterIsValid(t *testing.T) {
	for _, f := range []Filter{FilterDue, FilterAll, FilterNew, FilterLearning, FilterKnown} {
		if !f.IsValid() {
			t.Errorf("%q should be valid", f)
		}
	}
	for _, f := range []Filter{"", "soon", "DUE"} {
		if f.IsValid() {
			t.Errorf("%q should be invalid", f)
		}
	}
}

func TestFilterMatches(t *testing.T) {
	newCard := Card{Due: today}
	learning := Card{Interval: 1, Repetitions: 1, Due: today.AddDays(1)}
	known := Card{Interval: 6, Repetitions: 2, Due: today.AddDays(-1)}

	tests := []struct {
		filter Filter
		card   Card
		want   bool
	}{
		{FilterAll, learning, true},
		{FilterDue, newCard, true},
		{FilterDue, known, true},
		{FilterDue, learning, false},
		{FilterNew, newCard, true},
		{FilterNew, learning, false},
		{FilterLearning, learning, true},
		{FilterLearning, known, false},
		{FilterKnown, known, true},
		{FilterKnown, newCard, false},
	}
	for _, tt := range tests {
		if got := tt.filter.Matches(tt.card, today); got != tt.want {
			t.Errorf("%s.Matches(%+v) = %v, want %v", tt.filter, tt.card, got, tt.want)
		}
	}
}
