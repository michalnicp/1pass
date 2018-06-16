package main

import (
	"time"
)

// debounce returns a function that only calls f if the time elapsed since f was last called
// is greater than the duration.
func debounce(duration time.Duration, f func()) func() {
	var last time.Time

	return func() {
		t := time.Now()
		if t.After(last.Add(duration)) {
			last = t
			f()
		}
	}
}
