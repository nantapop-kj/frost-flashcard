package exporter

import (
	"bytes"
	"errors"
	"testing"

	"github.com/nantapop-kj/frost-flashcard/internal/importer"
	"github.com/nantapop-kj/frost-flashcard/internal/model"
)

var today = model.NewDate(2026, 1, 1)

func TestWriteCSV(t *testing.T) {
	var d model.Deck
	d.AddCard("猫", "แมว", "ねこ", today)
	d.AddCard("a, b", "say \"hi\"", "", today)

	var buf bytes.Buffer
	if err := WriteCSV(&buf, &d); err != nil {
		t.Fatal(err)
	}
	want := "\xEF\xBB\xBFword,meaning,reading\n猫,แมว,ねこ\n\"a, b\",\"say \"\"hi\"\"\",\n"
	if got := buf.String(); got != want {
		t.Errorf("csv =\n%q\nwant\n%q", got, want)
	}
}

func TestExportThenImportRoundTrip(t *testing.T) {
	var src model.Deck
	src.AddCard("猫", "แมว", "ねこ", today)
	src.AddCard("line\nbreak", "x", "", today)

	var buf bytes.Buffer
	if err := WriteCSV(&buf, &src); err != nil {
		t.Fatal(err)
	}
	rows, err := importer.ParseCSV(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}

	var dst model.Deck
	rep := importer.AddRows(&dst, rows, today)
	if rep != (importer.Report{Added: 2}) {
		t.Fatalf("report = %+v", rep)
	}
	for i := range src.Cards {
		s, d := src.Cards[i], dst.Cards[i]
		if s.Front != d.Front || s.Back != d.Back || s.Reading != d.Reading {
			t.Errorf("card %d = %+v, want %+v", i, d, s)
		}
	}
}

type failWriter struct{ after int }

func (w *failWriter) Write(p []byte) (int, error) {
	if w.after <= 0 {
		return 0, errors.New("disk full")
	}
	w.after--
	return len(p), nil
}

func TestWriteCSVReportsWriteErrors(t *testing.T) {
	var d model.Deck
	d.AddCard("a", "b", "", today)
	for _, after := range []int{0, 1} {
		if err := WriteCSV(&failWriter{after: after}, &d); err == nil {
			t.Errorf("after=%d: want error", after)
		}
	}
}
