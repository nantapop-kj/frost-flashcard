package model

import (
	"math/rand/v2"
	"slices"
	"testing"
)

func TestDeckAddCardAssignsIncreasingIDs(t *testing.T) {
	var d Deck
	a := d.AddCard("a", "1", "", today)
	b := d.AddCard("b", "2", "bee", today)

	if a != 1 || b != 2 || d.NextID != 2 || len(d.Cards) != 2 {
		t.Fatalf("ids = %d, %d; deck = %+v", a, b, d)
	}
	if d.Card(b).Reading != "bee" {
		t.Errorf("reading not stored: %+v", d.Card(b))
	}
}

func TestDeckIDsNotReusedAfterRemove(t *testing.T) {
	var d Deck
	d.AddCard("a", "1", "", today)
	id := d.AddCard("b", "2", "", today)
	d.RemoveCard(id)

	if next := d.AddCard("c", "3", "", today); next == id {
		t.Errorf("id %d was reused", id)
	}
}

func TestDeckUpdateCardKeepsProgress(t *testing.T) {
	var d Deck
	id := d.AddCard("a", "1", "", today)
	d.Card(id).Interval, d.Card(id).Repetitions = 6, 2

	if !d.UpdateCard(id, "A", "one", "ay") {
		t.Fatal("UpdateCard returned false")
	}
	c := d.Card(id)
	if c.Front != "A" || c.Back != "one" || c.Reading != "ay" {
		t.Errorf("text not updated: %+v", c)
	}
	if c.Interval != 6 || c.Repetitions != 2 {
		t.Errorf("progress lost: %+v", c)
	}
	if d.UpdateCard(99, "x", "y", "") {
		t.Error("UpdateCard on missing id should return false")
	}
}

func TestDeckRemoveCard(t *testing.T) {
	var d Deck
	a := d.AddCard("a", "1", "", today)
	b := d.AddCard("b", "2", "", today)
	c := d.AddCard("c", "3", "", today)

	if !d.RemoveCard(b) {
		t.Fatal("RemoveCard returned false")
	}
	if d.Card(b) != nil || d.Card(a) == nil || d.Card(c) == nil || len(d.Cards) != 2 {
		t.Errorf("cards after remove = %+v", d.Cards)
	}
	if d.RemoveCard(b) {
		t.Error("removing twice should return false")
	}
}

func TestDeckCardReturnsPointerIntoDeck(t *testing.T) {
	var d Deck
	id := d.AddCard("a", "1", "", today)
	d.Card(id).Interval = 9
	if d.Cards[0].Interval != 9 {
		t.Error("Card should return a pointer to the stored card, not a copy")
	}
	if d.Card(42) != nil {
		t.Error("Card(42) should be nil")
	}
}

func deckWithMixedCards() Deck {
	var d Deck
	d.AddCard("new", "", "", today)
	d.AddCard("learning", "", "", today)
	d.AddCard("known", "", "", today)
	d.AddCard("known-later", "", "", today)

	*d.Card(2) = Card{ID: 2, Interval: 1, Repetitions: 1, Due: today.AddDays(1)}
	*d.Card(3) = Card{ID: 3, Interval: 6, Repetitions: 2, Due: today}
	*d.Card(4) = Card{ID: 4, Interval: 15, Repetitions: 3, Due: today.AddDays(10)}
	return d
}

func TestDeckStats(t *testing.T) {
	d := deckWithMixedCards()
	want := Stats{Total: 4, DueToday: 2, New: 1, Learning: 1, Known: 2}
	if got := d.Stats(today); got != want {
		t.Errorf("Stats = %+v, want %+v", got, want)
	}
	if got := d.DueCount(today); got != 2 {
		t.Errorf("DueCount = %d, want 2", got)
	}
}

func TestDeckSessionIDsFilters(t *testing.T) {
	d := deckWithMixedCards()
	tests := []struct {
		filter Filter
		want   []uint32
	}{
		{FilterAll, []uint32{1, 2, 3, 4}},
		{FilterDue, []uint32{1, 3}},
		{FilterNew, []uint32{1}},
		{FilterLearning, []uint32{2}},
		{FilterKnown, []uint32{3, 4}},
	}
	for _, tt := range tests {
		got := d.SessionIDs(tt.filter, today, nil)
		slices.Sort(got)
		if !slices.Equal(got, tt.want) {
			t.Errorf("SessionIDs(%s) = %v, want %v", tt.filter, got, tt.want)
		}
	}
}

func TestDeckSessionIDsShuffleIsDeterministicWithSeed(t *testing.T) {
	var d Deck
	for range 20 {
		d.AddCard("x", "y", "", today)
	}
	a := d.SessionIDs(FilterAll, today, rand.New(rand.NewPCG(1, 2)))
	b := d.SessionIDs(FilterAll, today, rand.New(rand.NewPCG(1, 2)))
	if !slices.Equal(a, b) {
		t.Errorf("same seed gave different orders: %v vs %v", a, b)
	}
	if slices.IsSorted(a) {
		t.Errorf("20 cards came back unshuffled: %v", a)
	}
}

func TestDeckSessionIDsEmptyIsNotNil(t *testing.T) {
	var d Deck
	if ids := d.SessionIDs(FilterDue, today, nil); ids == nil {
		t.Error("SessionIDs should return an empty slice, not nil")
	}
}
