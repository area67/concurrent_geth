package concurrent

import "time"

const(
	MIN_DELAY = 50
	MAX_DELAY = 1000
)

var (
	NumThreads   = 4
)

func Backoff(delay int) int {
	time.Sleep(time.Duration(delay) * time.Nanosecond)
	if delay < MAX_DELAY {
		delay *= 2
	}
	return delay
}