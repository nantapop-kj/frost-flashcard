package model

import (
	"errors"
	"strings"
)

var (
	ErrNotFound  = errors.New("deck not found")
	ErrLastDeck  = errors.New("cannot delete the last remaining deck")
	ErrEmptyName = errors.New("deck name is empty")
)

type Library struct {
	Decks      []Deck `json:"decks"`
	Active     uint32 `json:"active"`
	NextDeckID uint32 `json:"next_deck_id"`
}

func NewLibrary() *Library {
	return LibraryFromDeck(Deck{})
}

func LibraryFromDeck(d Deck) *Library {
	d.ID = 1
	return &Library{Decks: []Deck{d}, Active: 1, NextDeckID: 1}
}

func (l *Library) ActiveDeck() *Deck {
	if d := l.deck(l.Active); d != nil {
		return d
	}

	l.Active = l.Decks[0].ID

	return &l.Decks[0]
}

func (l *Library) CreateDeck(name string) (uint32, error) {
	name, err := cleanName(name)
	if err != nil {
		return 0, err
	}
	l.NextDeckID++
	l.Decks = append(l.Decks, Deck{ID: l.NextDeckID, Name: name})
	l.Active = l.NextDeckID

	return l.NextDeckID, nil
}

func (l *Library) RenameDeck(id uint32, name string) error {
	name, err := cleanName(name)
	if err != nil {
		return err
	}
	d := l.deck(id)
	if d == nil {
		return ErrNotFound
	}
	d.Name = name

	return nil
}

func (l *Library) DeleteDeck(id uint32) error {
	i := l.index(id)
	if i < 0 {
		return ErrNotFound
	}
	if len(l.Decks) == 1 {
		return ErrLastDeck
	}
	l.Decks = append(l.Decks[:i], l.Decks[i+1:]...)
	if l.Active == id {
		l.Active = l.Decks[0].ID
	}
	return nil
}

func (l *Library) SetActive(id uint32) error {
	if l.index(id) < 0 {
		return ErrNotFound
	}
	l.Active = id

	return nil
}

func (l *Library) index(id uint32) int {
	for i := range l.Decks {
		if l.Decks[i].ID == id {
			return i
		}
	}

	return -1
}

func (l *Library) deck(id uint32) *Deck {
	if i := l.index(id); i >= 0 {
		return &l.Decks[i]
	}
	return nil
}

func cleanName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", ErrEmptyName
	}

	return name, nil
}
