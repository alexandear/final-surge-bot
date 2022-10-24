package bot

import (
	"time"
)

type RealClock struct{}

func NewClock() *RealClock {
	return &RealClock{}
}

func (c *RealClock) Now() time.Time {
	return time.Now()
}
