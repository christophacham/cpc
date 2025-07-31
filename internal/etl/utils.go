package etl

import (
	"time"
)

// now returns the current time - extracted for easier testing
func now() time.Time {
	return time.Now()
}