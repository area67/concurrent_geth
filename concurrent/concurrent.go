package concurrent

import (
	"math/rand"
	"sync"
	"time"
)

const(
	MIN_DELAY = 50
	MAX_DELAY = 1000
)

// TODO: This should probably be a constant, it was changed to var to allow for run-time changes for performance evaluation
var (
	NumThreads   = 4
)

func Backoff(delay int) int {

	time.Sleep(time.Duration(rand.Intn(delay)) * time.Nanosecond)
	if delay < MAX_DELAY {
		delay *= 2
	}
	return delay
}

// from https://gist.github.com/xtrcode/8fdffd4a9a036517fa217046f40c59c4
func CopySyncMap(m sync.Map) sync.Map {
	var cp sync.Map

	m.Range(func(k, v interface{}) bool {
		vm, ok := v.(sync.Map)
		if ok {
			cp.Store(k, CopySyncMap(vm))
		} else {
			cp.Store(k, v)
		}

		return true
	})

	return cp
}