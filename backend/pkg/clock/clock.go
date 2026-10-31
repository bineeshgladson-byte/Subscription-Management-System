// Package clock makes "now" injectable so billing can be tested (and demoed)
// for any date without waiting for the calendar.
package clock

import "time"

type Clock interface {
	Now() time.Time
	// Today returns midnight of the current business day in the business timezone.
	Today() time.Time
}

type Real struct{ Loc *time.Location }

func (r Real) Now() time.Time { return time.Now().In(r.Loc) }

func (r Real) Today() time.Time {
	n := r.Now()
	return time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, r.Loc)
}

// Fixed is for tests.
type Fixed struct{ T time.Time }

func (f Fixed) Now() time.Time { return f.T }

func (f Fixed) Today() time.Time {
	return time.Date(f.T.Year(), f.T.Month(), f.T.Day(), 0, 0, 0, 0, f.T.Location())
}
