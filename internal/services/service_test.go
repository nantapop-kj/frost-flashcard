package services

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/nantapop-kj/frost-flashcard/internal/dto"
	"github.com/nantapop-kj/frost-flashcard/internal/model"
	"github.com/nantapop-kj/frost-flashcard/internal/storage"
)

type fakePicker struct {
	path     string
	err      error
	gotTitle string
	gotName  string
}

func (p *fakePicker) PickImportFile(title string) (string, error) {
	p.gotTitle = title
	return p.path, p.err
}

func (p *fakePicker) PickExportFile(title, filename string) (string, error) {
	p.gotTitle, p.gotName = title, filename
	return p.path, p.err
}

func newTestService(t *testing.T) (*Service, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "flashcards.json")
	s := New(model.NewLibrary(), path, &fakePicker{})
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

func reload(t *testing.T, path string) *model.Library {
	t.Helper()
	lib, err := storage.LoadLibrary(path)
	if err != nil {
		t.Fatal(err)
	}
	return lib
}

func wantCode(t *testing.T, err error, code string) {
	t.Helper()
	if !dto.IsCode(err, code) {
		t.Errorf("err = %v, want code %s", err, code)
	}
}

func TestUpdateCardKeepsProgress(t *testing.T) {
	s, path := newTestService(t)
	id, _ := s.AddCard("a", "b", "")
	s.GradeCard(id, "good")

	if err := s.UpdateCard(id, " 猫 ", " cat ", " ねこ "); err != nil {
		t.Fatal(err)
	}
	c := reload(t, path).ActiveDeck().Card(id)
	if c.Front != "猫" || c.Back != "cat" || c.Reading != "ねこ" || c.Interval != 1 {
		t.Errorf("card = %+v", c)
	}

	wantCode(t, s.UpdateCard(id, "", "x", ""), dto.CodeEmptyFields)
	wantCode(t, s.UpdateCard(99, "x", "y", ""), dto.CodeNotFound)
}

func TestDeleteCard(t *testing.T) {
	s, path := newTestService(t)
	id, _ := s.AddCard("a", "b", "")

	if err := s.DeleteCard(id); err != nil {
		t.Fatal(err)
	}
	if n := len(reload(t, path).ActiveDeck().Cards); n != 0 {
		t.Errorf("cards on disk = %d, want 0", n)
	}
	wantCode(t, s.DeleteCard(id), dto.CodeNotFound)
}

func TestUpdateAndDeleteRollBackWhenSaveFails(t *testing.T) {
	s, _ := newTestService(t)
	id, _ := s.AddCard("a", "b", "")
	s.path = t.TempDir()

	wantCode(t, s.UpdateCard(id, "x", "y", ""), dto.CodeSaveFailed)
	wantCode(t, s.DeleteCard(id), dto.CodeSaveFailed)
	if cards := s.ListCards(); len(cards) != 1 || cards[0].Front != "a" {
		t.Errorf("cards = %+v, want unchanged", cards)
	}
}

func TestStats(t *testing.T) {
	s, _ := newTestService(t)
	a, _ := s.AddCard("a", "1", "")
	s.AddCard("b", "2", "")
	s.GradeCard(a, "good")

	got := s.Stats()
	want := model.Stats{Total: 2, DueToday: 1, New: 1, Learning: 1}
	if got != want {
		t.Errorf("Stats = %+v, want %+v", got, want)
	}
}

func TestStartSessionFilters(t *testing.T) {
	s, _ := newTestService(t)
	a, _ := s.AddCard("a", "1", "")
	s.AddCard("b", "2", "")
	s.GradeCard(a, "good")

	for filter, want := range map[string]int{"all": 2, "due": 1, "new": 1, "learning": 1, "known": 0} {
		cards, err := s.StartSession(filter)
		if err != nil || len(cards) != want {
			t.Errorf("%s: %d cards, err %v; want %d", filter, len(cards), err, want)
		}
	}
}

func TestDeckLifecycle(t *testing.T) {
	s, path := newTestService(t)
	s.AddCard("a", "b", "")

	id, err := s.CreateDeck("  Japanese ")
	if err != nil {
		t.Fatal(err)
	}
	view := s.ListDecks()
	if view.Active != id || len(view.Decks) != 2 || view.Decks[1] != (dto.DeckInfo{ID: id, Name: "Japanese"}) {
		t.Fatalf("ListDecks = %+v", view)
	}
	if view.Decks[0].Count != 1 || view.Decks[0].Due != 1 {
		t.Errorf("first deck = %+v", view.Decks[0])
	}
	if len(s.ListCards()) != 0 {
		t.Error("new deck should be active and empty")
	}

	if err := s.RenameDeck(id, "日本語"); err != nil {
		t.Fatal(err)
	}
	if err := s.SetActiveDeck(1); err != nil {
		t.Fatal(err)
	}
	lib := reload(t, path)
	if lib.Active != 1 || lib.Decks[1].Name != "日本語" {
		t.Errorf("saved library = %+v", lib)
	}

	if err := s.DeleteDeck(1); err != nil {
		t.Fatal(err)
	}
	if v := s.ListDecks(); v.Active != id || len(v.Decks) != 1 {
		t.Errorf("after delete = %+v", v)
	}
}

func TestDeckErrorsMapToCodes(t *testing.T) {
	s, _ := newTestService(t)

	_, err := s.CreateDeck(" ")
	wantCode(t, err, dto.CodeEmptyName)
	wantCode(t, s.RenameDeck(1, ""), dto.CodeEmptyName)
	wantCode(t, s.RenameDeck(9, "x"), dto.CodeNotFound)
	wantCode(t, s.SetActiveDeck(9), dto.CodeNotFound)
	wantCode(t, s.DeleteDeck(9), dto.CodeNotFound)
	wantCode(t, s.DeleteDeck(1), dto.CodeLastDeck)
}

func TestCreateDeckRollsBackWhenSaveFails(t *testing.T) {
	s, _ := newTestService(t)
	s.path = t.TempDir()

	_, err := s.CreateDeck("x")
	wantCode(t, err, dto.CodeSaveFailed)
	if v := s.ListDecks(); len(v.Decks) != 1 || v.Active != 1 {
		t.Errorf("decks = %+v, want unchanged", v)
	}
}

func TestImportFile(t *testing.T) {
	s, path := newTestService(t)
	s.AddCard("cat", "แมว", "")

	csv := filepath.Join(t.TempDir(), "words.csv")
	os.WriteFile(csv, []byte("word,meaning\nCAT,แมว\ndog,หมา\nbird,\n"), 0o644)
	picker := &fakePicker{path: csv}
	s.picker = picker

	rep, err := s.ImportFile("Pick a file")
	if err != nil {
		t.Fatal(err)
	}
	if rep == nil || rep.Added != 1 || rep.Duplicates != 1 || rep.Empty != 1 {
		t.Errorf("report = %+v", rep)
	}
	if picker.gotTitle != "Pick a file" {
		t.Errorf("title = %q", picker.gotTitle)
	}
	if n := len(reload(t, path).ActiveDeck().Cards); n != 2 {
		t.Errorf("cards on disk = %d, want 2", n)
	}
}

func TestImportFileCancelAndErrors(t *testing.T) {
	s, _ := newTestService(t)

	if rep, err := s.ImportFile(""); rep != nil || err != nil {
		t.Errorf("cancel = %v, %v; want nil, nil", rep, err)
	}

	s.picker = &fakePicker{err: errors.New("dialog broke")}
	_, err := s.ImportFile("")
	wantCode(t, err, dto.CodeReadFailed)

	dir := t.TempDir()
	bad := filepath.Join(dir, "words.csv")
	os.WriteFile(bad, []byte{0xA1, 0xB2}, 0o644)
	cases := map[string]string{
		filepath.Join(dir, "words.txt"): dto.CodeUnsupportedFile,
		bad:                             dto.CodeNotUTF8,
		filepath.Join(dir, "gone.csv"):  dto.CodeReadFailed,
	}
	for file, code := range cases {
		s.picker = &fakePicker{path: file}
		_, err := s.ImportFile("")
		wantCode(t, err, code)
	}
}

func TestImportFileRollsBackWhenSaveFails(t *testing.T) {
	s, _ := newTestService(t)
	csv := filepath.Join(t.TempDir(), "words.csv")
	os.WriteFile(csv, []byte("dog,หมา\n"), 0o644)
	s.picker = &fakePicker{path: csv}
	s.path = t.TempDir()

	_, err := s.ImportFile("")
	wantCode(t, err, dto.CodeSaveFailed)
	if n := len(s.ListCards()); n != 0 {
		t.Errorf("cards = %d, want 0", n)
	}
}

func TestExportCSV(t *testing.T) {
	s, _ := newTestService(t)
	s.RenameDeck(1, "JLPT N5/N4")
	s.AddCard("猫", "แมว", "ねこ")

	out := filepath.Join(t.TempDir(), "backup")
	picker := &fakePicker{path: out}
	s.picker = picker

	rep, err := s.ExportCSV("Save")
	if err != nil {
		t.Fatal(err)
	}
	if picker.gotName != "JLPT N5-N4.csv" {
		t.Errorf("suggested filename = %q", picker.gotName)
	}
	if rep == nil || rep.Count != 1 || rep.Path != out+".csv" {
		t.Fatalf("report = %+v", rep)
	}
	data, err := os.ReadFile(rep.Path)
	if err != nil || !strings.Contains(string(data), "猫,แมว,ねこ") {
		t.Errorf("file = %q, %v", data, err)
	}
}

func TestExportCSVCancelAndErrors(t *testing.T) {
	s, _ := newTestService(t)

	if rep, err := s.ExportCSV(""); rep != nil || err != nil {
		t.Errorf("cancel = %v, %v; want nil, nil", rep, err)
	}

	s.picker = &fakePicker{err: errors.New("dialog broke")}
	_, err := s.ExportCSV("")
	wantCode(t, err, dto.CodeWriteFailed)

	s.picker = &fakePicker{path: filepath.Join(t.TempDir(), "missing", "x.csv")}
	_, err = s.ExportCSV("")
	wantCode(t, err, dto.CodeWriteFailed)
}

func TestSafeFilename(t *testing.T) {
	for in, want := range map[string]string{"": "flashcards", "  ": "flashcards", "a/b:c": "a-b-c", "ภาษาไทย": "ภาษาไทย"} {
		if got := safeFilename(in); got != want {
			t.Errorf("safeFilename(%q) = %q, want %q", in, got, want)
		}
	}
}
