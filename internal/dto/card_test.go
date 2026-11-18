package dto

import (
	"testing"

	"github.com/nantapop-kj/frost-flashcard/internal/model"
)

func TestNewCardView(t *testing.T) {
	c := model.NewCard(3, "猫", "cat", model.NewDate(2026, 1, 2))
	c.Reading = "ねこ"

	got := NewCardView(c)
	want := CardView{
		ReviewCard: ReviewCard{ID: 3, Front: "猫", Back: "cat", Reading: "ねこ"},
		Due:        "2026-01-02",
		Status:     model.StatusNew,
	}
	if got != want {
		t.Errorf("NewCardView = %+v, want %+v", got, want)
	}
}
