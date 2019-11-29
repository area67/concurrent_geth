package correctness_tool

import (
	"container/list"
	"github.com/ethereum/go-ethereum/concurrent"
	"math"
	"sync"
	"sync/atomic"
)

type Verifier struct {
	methodCount		int32
	finalOutcome	bool
	threadLists		[]*list.List
	numTxnsAdded	int32
	GlobalCount		int32
	threadsDone 	[]int32
	isInBackground	bool
	done 			sync.WaitGroup
}

// constructor
func NewVerifier() *Verifier {
	lists :=  make([]*list.List, concurrent.NumThreads)

	for i := 0 ; i < concurrent.NumThreads; i++ {
		lists[i] = list.New()
	}
	threadGroup := make([]int32, concurrent.NumThreads)

	return  &Verifier{
		methodCount:    0,
		finalOutcome:   true,
		threadLists:    lists,
		numTxnsAdded:   0,
		GlobalCount:    0,
		threadsDone:    threadGroup,
		isInBackground:	false,
		done: 			sync.WaitGroup{},
	}
}

func (v *Verifier) ConcurrentVerify() {
	v.isInBackground = true
	go v.verify()
}

func (v *Verifier) LockFreeAddTxn(txData *TransactionData) {
	// could we get what we are looking for
	var res bool
	res = true
	method1 := NewMethod(-1, txData.addrSender, txData.addrReceiver, FIFO, PRODUCER, res, txData.amount)

	v.threadLists[txData.tId].PushBack(method1)
	index := int(atomic.AddInt32(&v.numTxnsAdded, 1) - 1)
	method1.id = index
}


func (v *Verifier) ThreadFinished(tid int) {
	v.threadsDone[tid] = DONE
}

func (v *Verifier) Verify() bool{
	v.verify()
	return v.finalOutcome
}

func (v *Verifier) WaitVerifier() bool {
	v.done.Wait()
	return v.finalOutcome
}

func (v *Verifier) verify() {
	v.done.Add(1)
	defer v.done.Done()

	var countIterated uint64 = 0
	methods := make(map[int]*Method)
	items := make(map[int]*Item)

	var itStart int

	stop := false
	//var oldMin int64
	for !stop{
		methods := make(map[int]*Method)
		items := make(map[int]*Item)
		stop = true

		// for each thread
		for tId := 0; tId < concurrent.NumThreads; tId++ {

			if v.isInBackground && atomic.LoadInt32(&v.threadsDone[tId]) == WORKING {
				stop = false
			}

			// TODO: This is where Christina has her thread.done() checking
			maxTxs := int(atomic.LoadInt32(&v.numTxnsAdded))
			for e := v.threadLists[tId].Front(); e != nil; e = e.Next() {
				// TODO: For concurrent execution can potentially store the list, atomic swap in a new list then evaluate.
				var m Method
				// line 1259
				m = *e.Value.(*Method)

				// checks if item is logically in the list
				if m.id == -1 && m.id < maxTxs {
					continue
				}

				// TODO: I think this can be cleaned up some. ill look at it later
				// Add the method and items to the map, both use m.id as the key
				methods[m.id] = &m

				item := NewItem(m.id)
				item.producer = m.id
				items[item.key] = item
			}
		}
		v.verifyCheckpoint(methods, items, &itStart, &countIterated, true)
	}
	v.verifyCheckpoint(methods, items, &itStart, &countIterated, false)
}

func (v *Verifier) verifyCheckpoint(methods map[int]*Method, items map[int]*Item, itStart *int, countIterated *uint64, resetItStart bool) {
	// TODO: is there a reason for inconsistency between int, int32, and int64?
	//		minor grievance, but it makes reading harder to understand
	v.methodCount = int32(len(methods))
	if v.methodCount != 0 {

		it := 0
		end := len(methods) - 1

		//TODO: corner case
		if *countIterated == 0 {
			resetItStart = false
		} else if it != end {
			*itStart = *itStart + 1
			it = *itStart
		}

		for ; it < len(methods); it++ {

			v.methodCount++

			// 532
			*itStart = it
			resetItStart = false
			*countIterated++

			// TODO: I'm a little confused by this block, it maps to line 557 in Christina's code
			// do what if method is a producer?
			if methods[it].types == PRODUCER {
				// set the itItems producer to it. indexes used as keys
				items[it].producer = it

				if items[it].status == ABSENT {

					// reset item parameters
					items[it].status = PRESENT
					// clear item's demote methods
				}

				items[it].addInt(1)
				if methods[it].semantics == FIFO {
					// TODO: This starts on line 568, ill come back to it and review
					for it0 := 0; it0 != it; it0++ {
						// starts at first index and works up to current index
						// increment global count for big Oh calculation
						v.GlobalCount++
						// serializability, correctness condition. 576 TODO
						if methods[it0].itemAddrS == methods[it].itemAddrS {
							//TODO: Ask christina why demoteMethods is never emptied
							if methods[it].requestAmnt.Cmp(methods[it0].requestAmnt) == LESS &&
									items[it0].status == PRESENT && methods[it0].semantics == FIFO {
								items[it0].promoteItems.Push(items[it].key)
								items[it].demote()

							} else if methods[it0].requestAmnt.Cmp(methods[it].requestAmnt) == LESS &&
									items[it0].status == PRESENT && methods[it0].semantics == FIFO {

								items[it].promoteItems.Push(items[it0].key)
								items[it0].demote()
							} else {
							}
						}
					}
				}
			}
		}

		if resetItStart {
			*itStart--
		}

		// verify sums
		outcome := v.handleNonceOrder(methods, items)
		for itVerify := 0; itVerify < len(items); itVerify++ {

			if items[itVerify].sum < 0 {
				outcome = false
			}else if (math.Ceil(items[itVerify].sum) + items[itVerify].sumR) < 0 {
				outcome = false
			}

			var n int
			//if items.items[itVerify].(*Item).sumF == 0 {
			if items[itVerify].sumF == 0 {
				n = 0
			} else {
				n = -1
			}

			if (math.Ceil(items[itVerify].sum)+items[itVerify].sumF)*float64(n) < 0 {
				outcome = false
			}
		}
		if outcome == true {
			v.finalOutcome = true
		} else {
			v.finalOutcome = false
		}
	}
}


func (v *Verifier) handleNonceOrder(methods map[int]*Method, items map[int]*Item) bool {
	result := true
	for it := 0; it < len(items); it++ {
		// Do what if method is a consumer
		if methods[it].status == true {
			// promote reads
			if items[it].sum > 0 {
				items[it].sumR = 0
			}

			items[it].subInt(1)
			if items[it].sum < 0 {
				// nonce ordering failed
				result = false
				v.finalOutcome = false
			}
			items[it].status = ABSENT

			for items[it].promoteItems.Len() != 0 {
				// TODO: Review this section in her code, it may diverge slightly in logic. Line 718
				// get the top item to promote from items's stack of items to promote
				itemPromote := items[it].promoteItems.Peek().(int)
				// promote item
				items[itemPromote].promote()
				// finished. pop and move to next item to promote
				items[it].promoteItems.Pop()
			}
		}
	}
	return result
}