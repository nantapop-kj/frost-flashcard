package importer

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/nantapop-kj/frost-flashcard/internal/model"
)

var today = model.NewDate(2026, 1, 1)

func TestParseCSVStripsBOMAndPadsShortRows(t *testing.T) {
	rows, err := ParseCSV([]byte("\xEF\xBB\xBF猫,แมว,ねこ\ndog,หมา\n\"a, b\",c,d,extra\n"))
	if err != nil {
		t.Fatal(err)
	}
	want := []Row{{"猫", "แมว", "ねこ"}, {"dog", "หมา", ""}, {"a, b", "c", "d"}}
	if len(rows) != len(want) {
		t.Fatalf("rows = %q, want %q", rows, want)
	}
	for i := range want {
		if rows[i] != want[i] {
			t.Errorf("row %d = %q, want %q", i, rows[i], want[i])
		}
	}
}

func TestParseCSVRejectsNonUTF8(t *testing.T) {
	if _, err := ParseCSV([]byte{0xA1, 0xB2, ',', 'x'}); !errors.Is(err, ErrNotUTF8) {
		t.Errorf("err = %v, want ErrNotUTF8", err)
	}
}

func TestReadRowsRejectsUnknownExtension(t *testing.T) {
	if _, err := ReadRows("words.txt"); !errors.Is(err, ErrUnsupported) {
		t.Errorf("err = %v, want ErrUnsupported", err)
	}
}

func TestReadRowsCSVFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "words.CSV")
	if err := os.WriteFile(path, []byte("word,meaning\ncat,แมว\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rows, err := ReadRows(path)
	if err != nil || len(rows) != 2 || rows[1] != (Row{"cat", "แมว", ""}) {
		t.Fatalf("ReadRows = %q, %v", rows, err)
	}
}

func TestReadRowsExcelFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "words.xlsx")
	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	f.SetSheetRow(sheet, "A1", &[]any{"単語", "意味", "読み"})
	f.SetSheetRow(sheet, "A2", &[]any{"猫", "cat", "ねこ"})
	if err := f.SaveAs(path); err != nil {
		t.Fatal(err)
	}

	rows, err := ReadRows(path)
	if err != nil || len(rows) != 2 || rows[1] != (Row{"猫", "cat", "ねこ"}) {
		t.Fatalf("ReadRows = %q, %v", rows, err)
	}
}

func TestReadRowsBrokenExcelFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "words.xlsx")
	if err := os.WriteFile(path, []byte("not a zip"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadRows(path); err == nil {
		t.Error("want an error for a broken xlsx file")
	}
}

func TestAddRowsSkipsHeaderBlanksAndDuplicates(t *testing.T) {
	var d model.Deck
	d.AddCard("Cat", "แมว", "", today)

	rows := []Row{
		{"Word", "Meaning", ""},
		{"dog", "หมา", ""},
		{" cat ", "แมว", ""},
		{"DOG", "หมา", ""},
		{"bird", "", ""},
		{"", "", ""},
		{" 猫 ", " ねこ ", " neko "},
	}
	rep := AddRows(&d, rows, today)

	if rep != (Report{Added: 2, Duplicates: 2, Empty: 1}) {
		t.Errorf("report = %+v", rep)
	}
	if len(d.Cards) != 3 || d.Cards[2] != func() model.Card {
		c := model.NewCard(3, "猫", "ねこ", today)
		c.Reading = "neko"
		return c
	}() {
		t.Errorf("cards = %+v", d.Cards)
	}
}

func TestAddRowsKeepsFirstRowWhenNotAHeader(t *testing.T) {
	var d model.Deck
	rep := AddRows(&d, []Row{{"apple", "แอปเปิล", ""}}, today)
	if rep.Added != 1 {
		t.Errorf("report = %+v, want first row imported", rep)
	}
}

func TestReadRowsMissingCSVFile(t *testing.T) {
	_, err := ReadRows(filepath.Join(t.TempDir(), "gone.csv"))
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("err = %v, want not-exist", err)
	}
}

func TestReadRowsExcelWithoutSheets(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.xlsx")
	if err := excelize.NewFile().SaveAs(src); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "nosheet.xlsx")
	rewriteZip(t, src, path, "xl/workbook.xml", func(b []byte) []byte {
		return regexp.MustCompile(`<sheets>.*</sheets>`).ReplaceAll(b, nil)
	})

	if _, err := ReadRows(path); !errors.Is(err, ErrNoSheet) {
		t.Errorf("err = %v, want ErrNoSheet", err)
	}
}

func rewriteZip(t *testing.T, src, dst, name string, edit func([]byte) []byte) {
	t.Helper()
	zr, err := zip.OpenReader(src)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		if f.Name == name {
			data = edit(data)
		}
		w, err := zw.Create(f.Name)
		if err != nil {
			t.Fatal(err)
		}
		w.Write(data)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestVocabTemplateImportsCleanly(t *testing.T) {
	rows, err := ReadRows(filepath.Join("..", "..", "assets", "vocab-template.xlsx"))
	if err != nil {
		t.Fatal(err)
	}
	var d model.Deck
	rep := AddRows(&d, rows, today)
	if rep.Added == 0 || rep.Duplicates != 0 || rep.Empty != 0 {
		t.Errorf("report = %+v, want rows added and nothing skipped", rep)
	}
	for _, c := range d.Cards {
		if c.Front == "word" {
			t.Error("header row was imported as a card")
		}
	}
}

func TestReadRowsExcelSheetCannotBeRead(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.xlsx")
	if err := excelize.NewFile().SaveAs(src); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "badname.xlsx")
	rewriteZip(t, src, path, "xl/workbook.xml", func(b []byte) []byte {
		return bytes.Replace(b, []byte(`name="Sheet1"`), []byte(`name="Bad:Name"`), 1)
	})

	_, err := ReadRows(path)
	if err == nil || errors.Is(err, ErrNoSheet) {
		t.Errorf("err = %v, want a sheet read error", err)
	}
}
