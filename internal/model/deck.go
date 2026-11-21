package model

import "math/rand/v2"

type Deck struct {
	ID     uint32 `json:"id"`
	Name   string `json:"name"`
	Cards  []Card `json:"cards"`
	NextID uint32 `json:"next_id"`
}

type Stats struct {
	Total    int `json:"total"`
	DueToday int `json:"due_today"`
	New      int `json:"new_cards"`
	Learning int `json:"learning"`
	Known    int `json:"known"`
}

func (d *Deck) AddCard(front, back, reading string, today Date) uint32 {
	d.NextID++
	c := NewCard(d.NextID, front, back, today)
	c.Reading = reading
	d.Cards = append(d.Cards, c)

	return d.NextID
}

func (d *Deck) UpdateCard(id uint32, front, back, reading string) bool {
	c := d.Card(id)
	if c == nil {
		return false
	}
	c.Front, c.Back, c.Reading = front, back, reading
	return true
}

func (d *Deck) RemoveCard(id uint32) bool {
	for i, c := range d.Cards {
		if c.ID == id {
			d.Cards = append(d.Cards[:i], d.Cards[i+1:]...)
			return true
		}
	}
	return false
}

func (d *Deck) Card(id uint32) *Card {
	for i := range d.Cards {
		if d.Cards[i].ID == id {
			return &d.Cards[i]
		}
	}
	return nil
}

func (d Deck) DueCount(today Date) int {
	return d.count(func(c Card) bool { return c.IsDue(today) })
}

func (d Deck) SessionIDs(filter Filter, today Date, rng *rand.Rand) []uint32 {
	ids := []uint32{}
	for _, c := range d.Cards {
		if filter.Matches(c, today) {
			ids = append(ids, c.ID)
		}
	}

	swap := func(i, j int) { ids[i], ids[j] = ids[j], ids[i] }
	if rng != nil {
		rng.Shuffle(len(ids), swap)
	} else {
		rand.Shuffle(len(ids), swap)
	}
	return ids
}

func (d Deck) Stats(today Date) Stats {
	return Stats{
		Total:    len(d.Cards),
		DueToday: d.DueCount(today),
		New:      d.countStatus(StatusNew),
		Learning: d.countStatus(StatusLearning),
		Known:    d.countStatus(StatusKnown),
	}
}

func (d Deck) countStatus(s Status) int {
	return d.count(func(c Card) bool { return c.Status() == s })
}

func (d Deck) count(match func(Card) bool) int {
	n := 0
	for _, c := range d.Cards {
		if match(c) {
			n++
		}
	}
	return n
}
