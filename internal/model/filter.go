package model

type Filter string

const (
	FilterDue      Filter = "due"
	FilterAll      Filter = "all"
	FilterNew      Filter = "new"
	FilterLearning Filter = "learning"
	FilterKnown    Filter = "known"
)

func (f Filter) IsValid() bool {
	switch f {
	case FilterDue, FilterAll, FilterNew, FilterLearning, FilterKnown:
		return true
	}
	return false
}

func (f Filter) Matches(c Card, today Date) bool {
	switch f {
	case FilterDue:
		return c.IsDue(today)
	case FilterAll:
		return true
	default:
		return Status(f) == c.Status()
	}
}
