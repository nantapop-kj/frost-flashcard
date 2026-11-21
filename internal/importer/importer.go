package importer

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/xuri/excelize/v2"

	"github.com/nantapop-kj/frost-flashcard/internal/model"
)

var (
	ErrUnsupported = errors.New("unsupported file type")
	ErrNotUTF8     = errors.New("csv file is not UTF-8")
	ErrNoSheet     = errors.New("excel file has no sheets")
)

type Report struct {
	Added      int `json:"added"`
	Duplicates int `json:"duplicates"`
	Empty      int `json:"empty"`
}

type Row [3]string

var headerWords = map[string]bool{
	"front": true, "word": true, "words": true, "term": true, "question": true, "vocabulary": true,
}

func ReadRows(path string) ([]Row, error) {
	switch ext := strings.ToLower(filepath.Ext(path)); ext {
	case ".csv":
		return readCSV(path)
	case ".xlsx", ".xlsm":
		return readExcel(path)
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnsupported, ext)
	}
}

func readExcel(path string) ([]Row, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("open excel file %s: %w", path, err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, ErrNoSheet
	}
	raw, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, fmt.Errorf("read sheet %q: %w", sheets[0], err)
	}

	return toRows(raw), nil
}

func readCSV(path string) ([]Row, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read csv file %s: %w", path, err)
	}

	return ParseCSV(data)
}

func ParseCSV(data []byte) ([]Row, error) {
	data = bytes.TrimPrefix(data, []byte("\xEF\xBB\xBF"))
	if !utf8.Valid(data) {
		return nil, ErrNotUTF8
	}

	r := csv.NewReader(bytes.NewReader(data))
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	raw, _ := r.ReadAll()

	return toRows(raw), nil
}

func toRows(raw [][]string) []Row {
	rows := make([]Row, 0, len(raw))
	for _, cells := range raw {
		var r Row
		copy(r[:], cells)
		rows = append(rows, r)
	}

	return rows
}

func AddRows(d *model.Deck, rows []Row, today model.Date) Report {
	var rep Report

	existing := make(map[string]struct{}, len(d.Cards))
	for _, c := range d.Cards {
		existing[key(c.Front)] = struct{}{}
	}

	for i, row := range rows {
		front, back, reading := strings.TrimSpace(row[0]), strings.TrimSpace(row[1]), strings.TrimSpace(row[2])

		if i == 0 && headerWords[strings.ToLower(front)] {
			continue
		}
		if front == "" && back == "" && reading == "" {
			continue
		}
		if front == "" || back == "" {
			rep.Empty++
			continue
		}
		k := key(front)
		if _, dup := existing[k]; dup {
			rep.Duplicates++
			continue
		}
		existing[k] = struct{}{}

		d.AddCard(front, back, reading, today)
		rep.Added++
	}

	return rep
}

func key(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
