package scheduler

import (
	"testing"

	"github.com/nantapop-kj/frost-flashcard/internal/model"
)

func day(d int) model.Date { return model.NewDate(2026, 1, d) }

func TestGoodAnswersGrowInterval(t *testing.T) {
	c := model.NewCard(1, "a", "b", day(1))
	steps := []struct {
		on       model.Date
		interval uint32
	}{{day(1), 1}, {day(2), 6}, {day(8), 15}}

	for _, s := range steps {
		Review(&c, Good, s.on)
		if c.Interval != s.interval {
			t.Fatalf("interval = %d, want %d", c.Interval, s.interval)
		}
	}
	if want := day(8).AddDays(15); !c.Due.Equal(want.Time) {
		t.Errorf("due = %v, want %v", c.Due, want)
	}
}

func TestAgainResetsProgress(t *testing.T) {
	c := model.NewCard(1, "a", "b", day(1))
	Review(&c, Good, day(1))
	Review(&c, Good, day(2))
	Review(&c, Again, day(8))

	if c.Repetitions != 0 || c.Interval != 1 {
		t.Errorf("reps=%d interval=%d, want 0 and 1", c.Repetitions, c.Interval)
	}
	if c.Ease >= model.DefaultEase {
		t.Errorf("ease = %v, want below %v", c.Ease, model.DefaultEase)
	}
}

func TestEaseNeverBelowMinimum(t *testing.T) {
	c := model.NewCard(1, "a", "b", day(1))
	for range 10 {
		Review(&c, Again, day(1))
	}
	if c.Ease != minEase {
		t.Errorf("ease = %v, want %v", c.Ease, minEase)
	}
}

func TestParseGrade(t *testing.T) {
	for _, s := range []string{"again", "hard", "good", "easy"} {
		if g, err := ParseGrade(s); err != nil || string(g) != s {
			t.Errorf("ParseGrade(%q) = %q, %v", s, g, err)
		}
	}
	if _, err := ParseGrade("perfect"); err == nil {
		t.Error("ParseGrade(perfect) should fail")
	}
}
