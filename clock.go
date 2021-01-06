package main

import (
	"time"
)

type RealClock struct {
}

func (c *RealClock) Now() time.Time {
	return time.Now()
}
