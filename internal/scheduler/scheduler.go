package scheduler

import (
	"fmt"
	"math"

	"github.com/nantapop-kj/frost-flashcard/internal/model"
)

const minEase = 1.3

type Grade string

const (
	Again Grade = "again"
	Hard  Grade = "hard"
	Good  Grade = "good"
	Easy  Grade = "easy"
)

var quality = map[Grade]float64{Again: 1, Hard: 3, Good: 4, Easy: 5}

func ParseGrade(s string) (Grade, error) {
	g := Grade(s)
	if _, ok := quality[g]; !ok {
		return "", fmt.Errorf("unknown grade %q (want again, hard, good or easy)", s)
	}
	return g, nil
}

func Review(c *model.Card, g Grade, today model.Date) {
	if g == Again {
		c.Repetitions = 0
		c.Interval = 1
	} else {
		switch c.Repetitions {
		case 0:
			c.Interval = 1
		case 1:
			c.Interval = 6
		default:
			c.Interval = uint32(math.Round(float64(c.Interval) * c.Ease))
		}
		c.Repetitions++
	}

	miss := 5 - quality[g]
	c.Ease = math.Max(c.Ease+0.1-miss*(0.08+miss*0.02), minEase)

	c.Due = today.AddDays(int(c.Interval))
}
