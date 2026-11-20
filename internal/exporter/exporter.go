package exporter

import (
	"encoding/csv"
	"io"

	"github.com/nantapop-kj/frost-flashcard/internal/model"
)

var header = []string{"word", "meaning", "reading"}

func WriteCSV(w io.Writer, d *model.Deck) error {
	if _, err := io.WriteString(w, "\xEF\xBB\xBF"); err != nil {
		return err
	}

	records := make([][]string, 0, len(d.Cards)+1)
	records = append(records, header)
	for _, c := range d.Cards {
		records = append(records, []string{c.Front, c.Back, c.Reading})
	}

	return csv.NewWriter(w).WriteAll(records)
}
