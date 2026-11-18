package services

import (
	"fmt"
	"strings"
	"sync"

	"github.com/nantapop-kj/frost-flashcard/internal/dto"
	"github.com/nantapop-kj/frost-flashcard/internal/model"
	"github.com/nantapop-kj/frost-flashcard/internal/scheduler"
	"github.com/nantapop-kj/frost-flashcard/internal/storage"
)

type Service struct {
	mu    sync.Mutex
	lib   *model.Library
	path  string
	today func() model.Date
}

func New(lib *model.Library, path string) *Service {
	return &Service{lib: lib, path: path, today: model.Today}
}

func (s *Service) AddCard(front, back, reading string) (uint32, error) {
	front, back, reading = strings.TrimSpace(front), strings.TrimSpace(back), strings.TrimSpace(reading)
	if front == "" || back == "" {
		return 0, &dto.Error{Code: dto.CodeEmptyFields, Detail: "front and back must not be blank"}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	deck := s.lib.ActiveDeck()
	id := deck.AddCard(front, back, reading, s.today())
	if err := s.save(); err != nil {
		deck.RemoveCard(id)
		return 0, err
	}
	return id, nil
}

func (s *Service) ListCards() []dto.CardView {
	s.mu.Lock()
	defer s.mu.Unlock()

	cards := s.lib.ActiveDeck().Cards
	out := make([]dto.CardView, 0, len(cards))
	for _, c := range cards {
		out = append(out, dto.NewCardView(c))
	}
	return out
}

func (s *Service) StartSession(filter string) ([]dto.ReviewCard, error) {
	f := model.Filter(filter)
	if !f.IsValid() {
		return nil, &dto.Error{Code: dto.CodeBadInput, Detail: fmt.Sprintf("unknown filter %q (want due, all, new, learning or known)", filter)}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	deck := s.lib.ActiveDeck()
	ids := deck.SessionIDs(f, s.today(), nil)
	out := make([]dto.ReviewCard, 0, len(ids))
	for _, id := range ids {
		if c := deck.Card(id); c != nil {
			out = append(out, dto.NewReviewCard(*c))
		}
	}
	return out, nil
}

func (s *Service) GradeCard(id uint32, grade string) (uint32, error) {
	g, err := scheduler.ParseGrade(grade)
	if err != nil {
		return 0, &dto.Error{Code: dto.CodeBadInput, Detail: err.Error()}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	c := s.lib.ActiveDeck().Card(id)
	if c == nil {
		return 0, &dto.Error{Code: dto.CodeNotFound, Detail: fmt.Sprintf("no card with id %d in the active deck", id)}
	}
	before := *c
	scheduler.Review(c, g, s.today())
	if err := s.save(); err != nil {
		*c = before
		return 0, err
	}
	return c.Interval, nil
}

func (s *Service) save() error {
	if err := storage.SaveLibrary(s.lib, s.path); err != nil {
		return &dto.Error{Code: dto.CodeSaveFailed, Detail: "changes were not saved: " + err.Error()}
	}
	return nil
}
