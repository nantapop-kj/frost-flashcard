package model

import (
	"encoding/json"
	"testing"
	"time"
)

func TestDateAddDaysCrossesMonthAndYear(t *testing.T) {
	got := NewDate(2025, time.December, 30).AddDays(3)
	if want := NewDate(2026, time.January, 2); !got.Equal(want.Time) {
		t.Errorf("AddDays = %v, want %v", got, want)
	}
}

func TestDateString(t *testing.T) {
	if got := NewDate(2026, time.March, 5).String(); got != "2026-03-05" {
		t.Errorf("String = %q", got)
	}
}

func TestTodayHasNoTimeOfDay(t *testing.T) {
	d := Today()
	if d.Hour() != 0 || d.Minute() != 0 || d.Second() != 0 || d.Nanosecond() != 0 || d.Location() != time.UTC {
		t.Errorf("Today = %v, want midnight UTC", d.Time)
	}
}

func TestDateJSONRoundTrip(t *testing.T) {
	in := NewDate(2026, time.February, 28)
	data, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `"2026-02-28"` {
		t.Errorf("Marshal = %s", data)
	}

	var out Date
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if !out.Equal(in.Time) {
		t.Errorf("Unmarshal = %v, want %v", out, in)
	}
}

func TestDateUnmarshalRejectsBadInput(t *testing.T) {
	for _, input := range []string{`"2026/02/28"`, `"2026-13-01"`, `""`, `20260228`, `null`} {
		var d Date
		if err := json.Unmarshal([]byte(input), &d); err == nil {
			t.Errorf("Unmarshal(%s) should fail, got %v", input, d)
		}
	}
}
