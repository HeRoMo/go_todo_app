package clock

import "time"

type Clocker interface {
	Now() time.Time
}

type RealClocker struct{}

func (r RealClocker) Now() time.Time {
	return time.Now()
}

type FixedClocker struct{}

func (r FixedClocker) Now() time.Time {
	return time.Date(2022, 8, 23, 19, 14, 22, 0, time.UTC)
}
