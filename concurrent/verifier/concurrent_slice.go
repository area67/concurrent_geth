package correctness_tool

import "sync"

type ConcurrentSlice struct {
	sync.RWMutex
	items []interface{}
	//items []ConcurrentSliceItem
}

// Concurrent slice item
type ConcurrentSliceItem struct {
	Index int
	Value interface{}
}

func (cs *ConcurrentSlice) Append(item interface{}) {
	cs.Lock()
	defer cs.Unlock()

	cs.items = append(cs.items, item)
}

func (cs *ConcurrentSlice) Iter() <-chan ConcurrentSliceItem {
	c := make(chan ConcurrentSliceItem)

	f := func() {
		cs.Lock()
		defer cs.Unlock()
		for index, value := range cs.items {
			c <- ConcurrentSliceItem{index, value}
		}
		close(c)
	}
	go f()

	return c
}