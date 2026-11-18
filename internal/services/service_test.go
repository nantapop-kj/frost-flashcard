package services

import (
	"path/filepath"
	"sync"
	"testing"

	"github.com/nantapop-kj/frost-flashcard/internal/dto"
	"github.com/nantapop-kj/frost-flashcard/internal/model"
	"github.com/nantapop-kj/frost-flashcard/internal/storage"
)

func newTestService(t *testing.T) (*Service, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "flashcards.json")
	s := New(model.NewLibrary(), path)
	s.today = func() model.Date { return model.NewDate(2026, 1, 1) }
	return s, path
}

func TestAddCardTrimsAndPersists(t *testing.T) {
	s, path := newTestService(t)

	id, err := s.AddCard("  猫 ", "แมว", " ねこ ")
	if err != nil {
		t.Fatal(err)
	}

	lib, err := storage.LoadLibrary(path)
	if err != nil {
		t.Fatal(err)
	}
	c := lib.ActiveDeck().Card(id)
	if c == nil || c.Front != "猫" || c.Reading != "ねこ" {
		t.Fatalf("saved card = %+v", c)
	}
}

func TestAddCardRejectsEmpty(t *testing.T) {
	s, _ := newTestService(t)
	if _, err := s.AddCard(" ", "x", ""); !dto.IsCode(err, dto.CodeEmptyFields) {
		t.Errorf("err = %v, want %s", err, dto.CodeEmptyFields)
	}
}

func TestStartSessionAndGrade(t *testing.T) {
	s, _ := newTestService(t)
	id, _ := s.AddCard("a", "b", "")

	cards, err := s.StartSession("due")
	if err != nil || len(cards) != 1 || cards[0].ID != id {
		t.Fatalf("session = %v, %v", cards, err)
	}

	days, err := s.GradeCard(id, "good")
	if err != nil || days != 1 {
		t.Fatalf("GradeCard = %d, %v; want 1", days, err)
	}

	cards, _ = s.StartSession("due")
	if len(cards) != 0 {
		t.Errorf("graded card still due: %v", cards)
	}
	if got := s.ListCards()[0].Status; got != model.StatusLearning {
		t.Errorf("status = %s, want learning", got)
	}
}

func TestBadInput(t *testing.T) {
	s, _ := newTestService(t)
	if _, err := s.StartSession("soon"); !dto.IsCode(err, dto.CodeBadInput) {
		t.Errorf("bad filter err = %v", err)
	}
	if _, err := s.GradeCard(1, "perfect"); !dto.IsCode(err, dto.CodeBadInput) {
		t.Errorf("bad grade err = %v", err)
	}
	if _, err := s.GradeCard(99, "good"); !dto.IsCode(err, dto.CodeNotFound) {
		t.Errorf("missing card err = %v", err)
	}
}

func TestGradeRollsBackWhenSaveFails(t *testing.T) {
	s, _ := newTestService(t)
	id, _ := s.AddCard("a", "b", "")
	s.path = t.TempDir()

	if _, err := s.GradeCard(id, "easy"); !dto.IsCode(err, dto.CodeSaveFailed) {
		t.Fatalf("err = %v, want %s", err, dto.CodeSaveFailed)
	}
	if c := s.lib.ActiveDeck().Card(id); c.Interval != 0 || c.Ease != model.DefaultEase {
		t.Errorf("card changed despite failed save: %+v", c)
	}
}

func TestConcurrentCalls(t *testing.T) {
	s, _ := newTestService(t)
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			id, _ := s.AddCard("a", "b", "")
			s.ListCards()
			s.StartSession("all")
			s.GradeCard(id, "good")
		})
	}
	wg.Wait()
	if n := len(s.ListCards()); n != 20 {
		t.Errorf("cards = %d, want 20", n)
	}
}

func TestAddCardRollsBackWhenSaveFails(t *testing.T) {
	s, _ := newTestService(t)
	s.path = t.TempDir()

	if _, err := s.AddCard("a", "b", ""); !dto.IsCode(err, dto.CodeSaveFailed) {
		t.Fatalf("err = %v, want %s", err, dto.CodeSaveFailed)
	}
	if n := len(s.ListCards()); n != 0 {
		t.Errorf("cards = %d, want 0 after failed save", n)
	}
}
