package services

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/nantapop-kj/frost-flashcard/internal/dto"
	"github.com/nantapop-kj/frost-flashcard/internal/exporter"
	"github.com/nantapop-kj/frost-flashcard/internal/importer"
	"github.com/nantapop-kj/frost-flashcard/internal/model"
	"github.com/nantapop-kj/frost-flashcard/internal/scheduler"
	"github.com/nantapop-kj/frost-flashcard/internal/storage"
)

type FilePicker interface {
	PickImportFile(title string) (string, error)
	PickExportFile(title, filename string) (string, error)
}

var knownErrors = []dto.Known{
	{Err: model.ErrNotFound, Code: dto.CodeNotFound},
	{Err: model.ErrLastDeck, Code: dto.CodeLastDeck},
	{Err: model.ErrEmptyName, Code: dto.CodeEmptyName},
	{Err: importer.ErrUnsupported, Code: dto.CodeUnsupportedFile},
	{Err: importer.ErrNotUTF8, Code: dto.CodeNotUTF8},
	{Err: importer.ErrNoSheet, Code: dto.CodeNoSheet},
}

func toError(err error, fallback string) error {
	return dto.FromError(err, fallback, knownErrors...)
}

type Service struct {
	mu     sync.Mutex
	lib    *model.Library
	path   string
	picker FilePicker
	today  func() model.Date
}

func New(lib *model.Library, path string, picker FilePicker) *Service {
	return &Service{lib: lib, path: path, picker: picker, today: model.Today}
}

func (s *Service) AddCard(front, back, reading string) (uint32, error) {
	front, back, reading, err := cleanCard(front, back, reading)
	if err != nil {
		return 0, err
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

func (s *Service) UpdateCard(id uint32, front, back, reading string) error {
	front, back, reading, err := cleanCard(front, back, reading)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return s.change(func(lib *model.Library) error {
		if !lib.ActiveDeck().UpdateCard(id, front, back, reading) {
			return cardNotFound(id)
		}

		return nil
	})
}

func (s *Service) DeleteCard(id uint32) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.change(func(lib *model.Library) error {
		if !lib.ActiveDeck().RemoveCard(id) {
			return cardNotFound(id)
		}

		return nil
	})
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

func (s *Service) Stats() model.Stats {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.lib.ActiveDeck().Stats(s.today())
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
		return 0, cardNotFound(id)
	}
	before := *c
	scheduler.Review(c, g, s.today())
	if err := s.save(); err != nil {
		*c = before
		return 0, err
	}

	return c.Interval, nil
}

func (s *Service) ListDecks() dto.DecksView {
	s.mu.Lock()
	defer s.mu.Unlock()

	today := s.today()
	view := dto.DecksView{Active: s.lib.ActiveDeck().ID, Decks: make([]dto.DeckInfo, 0, len(s.lib.Decks))}
	for _, d := range s.lib.Decks {
		view.Decks = append(view.Decks, dto.DeckInfo{
			ID:    d.ID,
			Name:  d.Name,
			Count: len(d.Cards),
			Due:   d.DueCount(today),
		})
	}

	return view
}

func (s *Service) CreateDeck(name string) (uint32, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var id uint32
	err := s.change(func(lib *model.Library) (err error) {
		id, err = lib.CreateDeck(name)
		return err
	})
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (s *Service) RenameDeck(id uint32, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.change(func(lib *model.Library) error { return lib.RenameDeck(id, name) })
}

func (s *Service) DeleteDeck(id uint32) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.change(func(lib *model.Library) error { return lib.DeleteDeck(id) })
}

func (s *Service) SetActiveDeck(id uint32) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.change(func(lib *model.Library) error { return lib.SetActive(id) })
}

func (s *Service) ImportFile(title string) (*importer.Report, error) {
	path, err := s.picker.PickImportFile(title)
	if err != nil {
		return nil, toError(err, dto.CodeReadFailed)
	}
	if path == "" {
		return nil, nil
	}

	rows, err := importer.ReadRows(path)
	if err != nil {
		return nil, toError(err, dto.CodeReadFailed)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	var rep importer.Report
	err = s.change(func(lib *model.Library) error {
		rep = importer.AddRows(lib.ActiveDeck(), rows, s.today())
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &rep, nil
}

func (s *Service) ExportCSV(title string) (*dto.ExportReport, error) {
	s.mu.Lock()
	name := s.lib.ActiveDeck().Name
	s.mu.Unlock()

	path, err := s.picker.PickExportFile(title, safeFilename(name)+".csv")
	if err != nil {
		return nil, toError(err, dto.CodeWriteFailed)
	}
	if path == "" {
		return nil, nil
	}
	if !strings.EqualFold(filepath.Ext(path), ".csv") {
		path += ".csv"
	}

	s.mu.Lock()
	deck := s.lib.ActiveDeck()
	count := len(deck.Cards)
	var buf bytes.Buffer
	err = exporter.WriteCSV(&buf, deck)
	s.mu.Unlock()
	if err != nil {
		return nil, toError(err, dto.CodeWriteFailed)
	}

	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return nil, toError(err, dto.CodeWriteFailed)
	}

	return &dto.ExportReport{
		Count: count,
		Path:  path,
	}, nil
}

func safeFilename(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "flashcards"
	}

	return strings.Map(func(r rune) rune {
		if r < ' ' || strings.ContainsRune(`/\:*?"<>|`, r) {
			return '-'
		}
		return r
	}, name)
}

func (s *Service) change(fn func(lib *model.Library) error) error {
	snapshot := s.lib.Clone()
	err := fn(s.lib)
	if err == nil {
		err = s.save()
	}
	if err != nil {
		*s.lib = *snapshot
		return toError(err, dto.CodeBadInput)
	}

	return nil
}

func (s *Service) save() error {
	if err := storage.SaveLibrary(s.lib, s.path); err != nil {
		return &dto.Error{Code: dto.CodeSaveFailed, Detail: "changes were not saved: " + err.Error()}
	}

	return nil
}

func cleanCard(front, back, reading string) (string, string, string, error) {
	front, back, reading = strings.TrimSpace(front), strings.TrimSpace(back), strings.TrimSpace(reading)
	if front == "" || back == "" {
		return "", "", "", &dto.Error{
			Code:   dto.CodeEmptyFields,
			Detail: "front and back must not be blank",
		}
	}

	return front, back, reading, nil
}

func cardNotFound(id uint32) error {
	return &dto.Error{
		Code:   dto.CodeNotFound,
		Detail: fmt.Sprintf("no card with id %d in the active deck", id),
	}
}
