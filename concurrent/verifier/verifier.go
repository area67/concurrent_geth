package correctness_tool

import (
	"container/list"
	"github.com/ethereum/go-ethereum/concurrent"
	"github.com/golang-collections/collections/stack"
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
	done 			sync.WaitGroup
}

// constructor
func NewVerifier() *Verifier {
	lists :=  make([]*list.List, concurrent.NumThreads)

	for i := 0 ; i < concurrent.NumThreads; i++ {
		lists[i] = list.New()
	}
	threadGroup := make([]int32, concurrent.NumThreads)

	verifierDone := sync.WaitGroup{}
	verifierDone.Add(1)
	return  &Verifier{
		methodCount:    0,
		finalOutcome:   true,
		threadLists:    lists,
		numTxnsAdded:   0,
		GlobalCount:    0,
		threadsDone:    threadGroup,
		done: 			verifierDone,
	}
}

func (v *Verifier) ConcurrentVerify() {
	go v.verify()
}

func (v *Verifier) LockFreeAddTxn(txData *TransactionData) {
	// could we get what we are looking for
	var res bool
	res = true
	method1 := NewMethod(-1, txData.addrSender, txData.addrReceiver, FIFO, PRODUCER, res, txData.amount)

	v.threadLists[txData.tId].PushBack(method1)
	index := int(atomic.AddInt32(&v.numTxnsAdded, 1) - 1)
	v.threadLists[txData.tId].Back().Value.(*Method).id = index
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
	defer v.done.Done()

	var countIterated uint64 = 0
	//methods := make([]Method, 0)
	methods := make(map[int]*Method)
	//items := make([]Item, 0, v.txnCtr * 2)
	items := make(map[int]*Item)
	//it := make([]int, concurrent.NumThreads, concurrent.NumThreads)
	var itStart int

	stop := false
	//var oldMin int64
	for !stop{
		stop = true

		// for each thread
		for tId := 0; tId < concurrent.NumThreads; tId++ {

			if atomic.LoadInt32(&v.threadsDone[tId]) == WORKING {
				stop = false
			}

			// TODO: This is where Christina has her thread.done() checking
			// can maybe use the shutdown function here, this object may need a wait group to signal when verify has finished

			// iterate thorough each thread's methods
			//for itCount[i] < v.threadListsSize[i].Load() {
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

	var stackConsumer = stack.New()      // stack of map[int64]*Item
	var stackFinishedMethods stack.Stack // stack of map[int64]*Method
	var stackFailed stack.Stack          // stack of map[int64]*Item

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
					items[it].demoteMethods.Init()
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
							if methods[it].requestAmnt.Cmp(methods[it0].requestAmnt) == LESS && items[it0].status == PRESENT && methods[it0].semantics == FIFO {
								items[it0].promoteItems.Push(items[it].key)
								items[it].demote()
								items[it].demoteMethods.PushBack(methods[it0])

							} else if methods[it0].requestAmnt.Cmp(methods[it].requestAmnt) == LESS && items[it0].status == PRESENT && methods[it0].semantics == FIFO {
								items[it].promoteItems.Push(items[it0].key)
								items[it0].demote()
								items[it0].demoteMethods.PushBack(methods[it])
							}
						}
					}
				}
			}

			if methods[it].types == CONSUMER {
				// Do what if method is a consumer
				if methods[it].status == true {
					// promote reads
					if items[it].sum > 0 {
						items[it].sumR = 0
					}
					//println("251")
					// line 637
					// sub on consumer?
					items[it].subInt(1)
					items[it].status = ABSENT

					stackConsumer.Push(it)
					stackFinishedMethods.Push(it)

					if items[it].producer != end {
						stackFinishedMethods.Push(items[it].producer)
					}
				} else {
					v.handleFailedConsumer(methods, items, it, &stackFailed)
				}
			}
		}
		if resetItStart {
			*itStart--
		}

		for stackConsumer.Len() != 0 {

			// get next item index to whose demoted items need to be promoted
			// itTop not a good var name in this case
			itTop, _ := stackConsumer.Peek().(int)

			// 717

			for items[itTop].promoteItems.Len() != 0 {
				// TODO: Review this section in her code, it may diverge slightly in logic. Line 718
				// get the top item to promote from items's stack of items to promote
				itemPromote := items[itTop].promoteItems.Peek().(int)
				// promote item
				items[itemPromote].promote()
				// finished. pop and move to next item to promote
				items[itTop].promoteItems.Pop()
			}
			stackConsumer.Pop()
		}

		for stackFailed.Len() != 0 {

			itTop := stackFailed.Peek().(int)

			//if items.items[itTop].(*Item).status == PRESENT {
			if items[itTop].status == PRESENT {
				items[itTop].demoteFailed()
			}
			stackFailed.Pop()

		}

		// remove methods from
		for stackFinishedMethods.Len() != 0 {
			// TODO: may need to remove items from methods. if we want to delete need to convert methods and items to map
			//delete(methods, key) // problem: keys are indexes which would change if we remove items from slice
			stackFinishedMethods.Pop()
		}

		// verify sums
		outcome := true

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

func (v *Verifier) handleFailedConsumer(methods map[int]*Method, items map[int]*Item, it int,  stackFailed *stack.Stack) {
	for it0 := 0; it0 != it; it0++ {
		// serializability
		if methods[it0].itemAddrS == methods[it].itemAddrS {
			if methods[it].requestAmnt.Cmp(methods[it0].requestAmnt) == LESS &&
				items[it0].status == PRESENT && methods[it0].semantics == FIFO {

			} else if methods[it0].requestAmnt.Cmp(methods[it].requestAmnt) == LESS &&
				items[it0].status == PRESENT && methods[it0].semantics == FIFO {

			}
		}

		if methods[it].itemAddrS == methods[it0].itemAddrS {
			itemItr0 := items[it0].key

			//if methods[it0].types == PRODUCER &&
			if methods[it0].types == CONSUMER &&
				items[it].status == PRESENT &&
				methods[it0].semantics == FIFO ||
				methods[it].itemAddrS == methods[it0].itemAddrS {
				stackFailed.Push(itemItr0)
			}
		}
	}
}
