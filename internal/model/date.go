package model

import (
	"encoding/json"
	"fmt"
	"time"
)

type Date struct{ time.Time }

const dateLayout string = "2006-01-02"

func NewDate(year int, month time.Month, day int) Date {
	return Date{time.Date(year, month, day, 0, 0, 0, 0, time.UTC)}
}

func Today() Date {
	now := time.Now()
	return NewDate(now.Year(), now.Month(), now.Day())
}

func (d Date) AddDays(n int) Date {
	return Date{d.Time.AddDate(0, 0, n)}
}

func (d Date) String() string {
	return d.Format(dateLayout)
}

func (d Date) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (d *Date) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	t, err := time.Parse(dateLayout, s)
	if err != nil {
		return fmt.Errorf("date %q is not in YYYY-MM-DD format: %w", s, err)
	}
	*d = Date{t}
	return nil
}
