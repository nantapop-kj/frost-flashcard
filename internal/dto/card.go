package dto

import "github.com/nantapop-kj/frost-flashcard/internal/model"

type ReviewCard struct {
	ID      uint32 `json:"id"`
	Front   string `json:"front"`
	Back    string `json:"back"`
	Reading string `json:"reading"`
}

type CardView struct {
	ReviewCard
	Due    string       `json:"due"`
	Status model.Status `json:"status"`
}

func NewReviewCard(c model.Card) ReviewCard {
	return ReviewCard{ID: c.ID, Front: c.Front, Back: c.Back, Reading: c.Reading}
}

func NewCardView(c model.Card) CardView {
	return CardView{ReviewCard: NewReviewCard(c), Due: c.Due.String(), Status: c.Status()}
}
