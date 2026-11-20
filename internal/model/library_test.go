package model

import (
	"errors"
	"testing"
)

func TestNewLibraryHasOneActiveDeck(t *testing.T) {
	l := NewLibrary()
	if len(l.Decks) != 1 || l.Active != 1 || l.ActiveDeck().ID != 1 {
		t.Errorf("NewLibrary = %+v", l)
	}
}

func TestLibraryFromDeckKeepsCards(t *testing.T) {
	var d Deck
	d.ID = 42
	d.AddCard("a", "b", "", today)

	l := LibraryFromDeck(d)
	if l.ActiveDeck().ID != 1 || len(l.ActiveDeck().Cards) != 1 {
		t.Errorf("LibraryFromDeck = %+v", l)
	}
}

func TestCreateDeckTrimsNameAndActivates(t *testing.T) {
	l := NewLibrary()
	id, err := l.CreateDeck("  Japanese  ")
	if err != nil {
		t.Fatal(err)
	}
	if id != 2 || l.Active != id || l.ActiveDeck().Name != "Japanese" {
		t.Errorf("after CreateDeck: id=%d %+v", id, l)
	}
}

func TestCreateDeckRejectsBlankName(t *testing.T) {
	l := NewLibrary()
	if _, err := l.CreateDeck(" \t "); !errors.Is(err, ErrEmptyName) {
		t.Errorf("err = %v, want ErrEmptyName", err)
	}
	if len(l.Decks) != 1 {
		t.Error("blank name should not create a deck")
	}
}

func TestRenameDeck(t *testing.T) {
	l := NewLibrary()
	if err := l.RenameDeck(1, " Words "); err != nil || l.Decks[0].Name != "Words" {
		t.Errorf("rename: err=%v name=%q", err, l.Decks[0].Name)
	}
	if err := l.RenameDeck(1, ""); !errors.Is(err, ErrEmptyName) {
		t.Errorf("blank rename err = %v", err)
	}
	if err := l.RenameDeck(99, "x"); !errors.Is(err, ErrNotFound) {
		t.Errorf("missing deck err = %v", err)
	}
}

func TestDeleteDeck(t *testing.T) {
	l := NewLibrary()
	id, _ := l.CreateDeck("Second")

	if err := l.DeleteDeck(id); err != nil {
		t.Fatal(err)
	}
	if len(l.Decks) != 1 || l.Active != 1 {
		t.Errorf("deleting the active deck should fall back to the first: %+v", l)
	}
	if err := l.DeleteDeck(1); !errors.Is(err, ErrLastDeck) {
		t.Errorf("deleting last deck err = %v", err)
	}
	if err := l.DeleteDeck(99); !errors.Is(err, ErrNotFound) {
		t.Errorf("missing deck err = %v", err)
	}
}

func TestDeleteInactiveDeckKeepsActive(t *testing.T) {
	l := NewLibrary()
	second, _ := l.CreateDeck("Second")
	if err := l.DeleteDeck(1); err != nil {
		t.Fatal(err)
	}
	if l.Active != second {
		t.Errorf("active = %d, want %d", l.Active, second)
	}
}

func TestDeckIDsNotReusedAfterDelete(t *testing.T) {
	l := NewLibrary()
	id, _ := l.CreateDeck("Second")
	l.DeleteDeck(id)
	if next, _ := l.CreateDeck("Third"); next == id {
		t.Errorf("deck id %d was reused", id)
	}
}

func TestSetActive(t *testing.T) {
	l := NewLibrary()
	l.CreateDeck("Second")
	if err := l.SetActive(1); err != nil || l.Active != 1 {
		t.Errorf("SetActive(1): err=%v active=%d", err, l.Active)
	}
	if err := l.SetActive(99); !errors.Is(err, ErrNotFound) || l.Active != 1 {
		t.Errorf("SetActive(99): err=%v active=%d", err, l.Active)
	}
}

func TestActiveDeckRepairsDanglingActiveID(t *testing.T) {
	l := NewLibrary()
	l.Active = 99
	if d := l.ActiveDeck(); d.ID != 1 || l.Active != 1 {
		t.Errorf("ActiveDeck = %d, Active = %d; want 1", d.ID, l.Active)
	}
}

func TestCloneIsIndependent(t *testing.T) {
	l := NewLibrary()
	l.ActiveDeck().AddCard("a", "b", "", today)

	c := l.Clone()
	c.ActiveDeck().Cards[0].Front = "changed"
	c.ActiveDeck().Name = "renamed"
	c.CreateDeck("second")

	if l.ActiveDeck().Cards[0].Front != "a" || l.ActiveDeck().Name != "" || len(l.Decks) != 1 {
		t.Errorf("original changed through clone: %+v", l)
	}
}
