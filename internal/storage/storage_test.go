package storage

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nantapop-kj/frost-flashcard/internal/model"
)

var today = model.NewDate(2026, 1, 1)

func writeFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "flashcards.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestSaveThenLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "flashcards.json")
	lib := model.NewLibrary()
	if _, err := lib.CreateDeck("Japanese"); err != nil {
		t.Fatal(err)
	}
	id := lib.ActiveDeck().AddCard("猫", "cat", "ねこ", today)
	lib.ActiveDeck().Card(id).Ease = 2.36

	if err := SaveLibrary(lib, path); err != nil {
		t.Fatal(err)
	}
	got, err := LoadLibrary(path)
	if err != nil {
		t.Fatal(err)
	}

	d := got.ActiveDeck()
	c := d.Card(id)
	if len(got.Decks) != 2 || got.NextDeckID != 2 || d.Name != "Japanese" || d.NextID != 1 {
		t.Errorf("library = %+v", got)
	}
	if c == nil || c.Front != "猫" || c.Reading != "ねこ" || c.Ease != 2.36 || !c.Due.Equal(today.Time) {
		t.Errorf("card = %+v", c)
	}
}

func TestLoadMissingFileReturnsNewLibrary(t *testing.T) {
	lib, err := LoadLibrary(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(lib.Decks) != 1 || len(lib.ActiveDeck().Cards) != 0 {
		t.Errorf("lib = %+v", lib)
	}
}

func TestLoadLegacySingleDeckFile(t *testing.T) {
	path := writeFile(t, `{"cards":[{"id":3,"front":"cat","back":"แมว","interval":0,"ease":2.5,"repetitions":0,"due":"2026-01-01"}],"next_id":3}`)

	lib, err := LoadLibrary(path)
	if err != nil {
		t.Fatal(err)
	}
	d := lib.ActiveDeck()
	if len(lib.Decks) != 1 || d.ID != 1 || d.NextID != 3 || len(d.Cards) != 1 {
		t.Fatalf("migrated = %+v", lib)
	}
	if c := d.Card(3); c == nil || c.Front != "cat" || c.Reading != "" {
		t.Errorf("card = %+v", c)
	}
}

func TestLoadEmptyDecksReturnsNewLibrary(t *testing.T) {
	lib, err := LoadLibrary(writeFile(t, `{"decks":[],"active":5}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(lib.Decks) != 1 || lib.Active != 1 {
		t.Errorf("lib = %+v", lib)
	}
}

func TestLoadRejectsBadFiles(t *testing.T) {
	tests := map[string]string{
		"not json":         `this is not json`,
		"bad legacy cards": `{"cards":"oops"}`,
		"bad decks":        `{"decks":{"id":1}}`,
		"bad date":         `{"decks":[{"id":1,"cards":[{"id":1,"due":"01/01/2026"}]}]}`,
	}
	for name, content := range tests {
		if _, err := LoadLibrary(writeFile(t, content)); err == nil {
			t.Errorf("%s: expected an error so user data is not overwritten", name)
		}
	}
}

func TestLoadUnreadablePath(t *testing.T) {
	if _, err := LoadLibrary(t.TempDir()); err == nil {
		t.Error("loading a directory should fail")
	}
}

func TestSaveCreatesFolderAndLeavesNoTempFile(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "app")
	path := filepath.Join(dir, "flashcards.json")

	if err := SaveLibrary(model.NewLibrary(), path); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "flashcards.json" {
		t.Errorf("folder contents = %v", entries)
	}
}

func TestSaveOverwritesExistingFile(t *testing.T) {
	path := writeFile(t, `{"decks":[{"id":1,"name":"old"}],"active":1}`)
	lib := model.NewLibrary()
	lib.Decks[0].Name = "new"

	if err := SaveLibrary(lib, path); err != nil {
		t.Fatal(err)
	}
	got, err := LoadLibrary(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Decks[0].Name != "new" {
		t.Errorf("name = %q, want new", got.Decks[0].Name)
	}
}

func TestSaveFailsWhenFolderIsAFile(t *testing.T) {
	blocker := writeFile(t, "")
	assertSaveFails(t, model.NewLibrary(), filepath.Join(blocker, "flashcards.json"), "create data folder")
}

func assertSaveFails(t *testing.T, lib *model.Library, path, wantMsg string) {
	t.Helper()
	err := SaveLibrary(lib, path)
	if err == nil {
		t.Fatalf("SaveLibrary(%s) succeeded, want error containing %q", path, wantMsg)
	}
	if !strings.Contains(err.Error(), wantMsg) {
		t.Errorf("err = %v, want it to contain %q", err, wantMsg)
	}
}

func TestSaveFailsWhenDataCannotBeEncoded(t *testing.T) {
	lib := model.NewLibrary()
	id := lib.ActiveDeck().AddCard("a", "b", "", today)
	lib.ActiveDeck().Card(id).Ease = math.NaN()

	path := filepath.Join(t.TempDir(), "flashcards.json")
	assertSaveFails(t, lib, path, "convert flashcards to JSON")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("nothing should be written when encoding fails")
	}
}

func TestSaveFailsWhenTempFileCannotBeWritten(t *testing.T) {
	path := filepath.Join(t.TempDir(), "flashcards.json")
	if err := os.Mkdir(path+".tmp", 0o755); err != nil {
		t.Fatal(err)
	}
	assertSaveFails(t, model.NewLibrary(), path, "write temporary file")
}

func TestSaveFailsWhenTargetCannotBeReplaced(t *testing.T) {
	path := filepath.Join(t.TempDir(), "flashcards.json")
	if err := os.MkdirAll(filepath.Join(path, "keep"), 0o755); err != nil {
		t.Fatal(err)
	}
	assertSaveFails(t, model.NewLibrary(), path, "replace "+path)
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Error("temporary file should be removed when replacing fails")
	}
}
