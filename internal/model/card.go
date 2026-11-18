package model

const DefaultEase = 2.5

type Status string

const (
	StatusNew      Status = "new"
	StatusLearning Status = "learning"
	StatusKnown    Status = "known"
)

type Card struct {
	ID          uint32  `json:"id"`
	Front       string  `json:"front"`
	Back        string  `json:"back"`
	Reading     string  `json:"reading,omitempty"`
	Interval    uint32  `json:"interval"`
	Ease        float64 `json:"ease"`
	Repetitions uint32  `json:"repetitions"`
	Due         Date    `json:"due"`
}

func NewCard(id uint32, front, back string, today Date) Card {
	return Card{
		ID:    id,
		Front: front,
		Back:  back,
		Ease:  DefaultEase,
		Due:   today,
	}
}

func (c Card) Status() Status {
	switch {
	case c.Interval == 0:
		return StatusNew
	case c.Repetitions >= 2:
		return StatusKnown
	default:
		return StatusLearning
	}
}

func (c Card) IsDue(today Date) bool {
	return !c.Due.After(today.Time)
}
