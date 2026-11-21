package model

import "testing"

var today = NewDate(2026, 1, 10)

func TestNewCardDefaults(t *testing.T) {
	c := NewCard(7, "cat", "แมว", today)
	if c.ID != 7 || c.Front != "cat" || c.Back != "แมว" {
		t.Errorf("fields = %+v", c)
	}
	if c.Ease != DefaultEase || c.Interval != 0 || c.Repetitions != 0 {
		t.Errorf("progress = %+v, want fresh card", c)
	}
	if !c.Due.Equal(today.Time) {
		t.Errorf("due = %v, want %v", c.Due, today)
	}
}

func TestCardStatus(t *testing.T) {
	tests := []struct {
		name        string
		interval    uint32
		repetitions uint32
		want        Status
	}{
		{"never reviewed", 0, 0, StatusNew},
		{"forgotten", 1, 0, StatusLearning},
		{"one success", 1, 1, StatusLearning},
		{"two successes", 6, 2, StatusKnown},
		{"many successes", 95, 5, StatusKnown},
	}
	for _, tt := range tests {
		c := Card{Interval: tt.interval, Repetitions: tt.repetitions}
		if got := c.Status(); got != tt.want {
			t.Errorf("%s: Status = %s, want %s", tt.name, got, tt.want)
		}
	}
}

func TestCardIsDue(t *testing.T) {
	tests := []struct {
		due  Date
		want bool
	}{
		{today.AddDays(-1), true},
		{today, true},
		{today.AddDays(1), false},
	}
	for _, tt := range tests {
		c := Card{Due: tt.due}
		if got := c.IsDue(today); got != tt.want {
			t.Errorf("due %v: IsDue = %v, want %v", tt.due, got, tt.want)
		}
	}
}
